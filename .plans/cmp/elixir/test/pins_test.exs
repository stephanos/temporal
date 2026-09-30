defmodule Temporal.Feature.PinsTest do
  @moduledoc """
  What the two Models say.

  The claims below are about the tables, because the tables are what the search, the Behavior
  Fingerprint and Contract lowering read. A `case` arm that stopped saying what it says would fail
  here. Three tiers, like the Lean `#guard` pins but split by where Elixir can run them:

    * `static_assert/1` is decided when ExUnit compiles this file, before any test runs;
    * everything the Models' `@before_compile` hooks already checked (coverage, refinement,
      admission) failed `mix compile` if it was wrong, so the tests here pin its answers;
    * queries, the vacuity rule and the agreement of IR and compiled code run as tests.
  """

  use Umpire.Case

  alias Temporal.Feature.Nexus.Caller, as: Nexus
  alias Temporal.Feature.StandaloneActivity, as: Activity
  alias Umpire.{Domain, IR, Refinement, Search, Step, Table}

  # Decided at this file's compile time: twelve phases, three attempt counts and three deadlines.
  static_assert Activity.ProtocolState.size() == 12 * 3 * 2 * 2 * 2
  static_assert Nexus.ProtocolState.size() == 8 * 3 * 2 * 2 * 2

  # A state written the way a reader names one: the phase, and every other field at the value
  # the entity begins with.
  defp at(state_module, phase), do: Domain.start(state_module, phase)

  defp verdict(%Refinement.Result{rows: rows}, from, class) do
    Enum.find_value(rows, fn {row, verdict} -> row.from == from and row.class == class and verdict end)
  end

  defp action_classes(table), do: length(table.classes)

  describe "Nexus caller" do
    alias Nexus.{NexusProduct, NexusProtocol, ProductState, ProductPhase}

    # Six phases, and the four the design ends on.
    test "the product machine has 6 states and 4 ends" do
      assert length(NexusProduct.table().states) == 6
      assert length(NexusProduct.table().ends) == 4
    end

    # Every action class the machine steps on: six replies, three resolutions, the two faults it
    # cannot see, and the one timer.
    test "the product machine has 12 action classes" do
      assert action_classes(NexusProduct.table()) == 6 + 3 + 1 + 1 + 1
    end

    # A retryable handler error is invisible here: it is the protocol machine that backs off.
    test "a retryable handler error from scheduled is not enabled on the product" do
      assert NexusProduct.handler_reply(%ProductState{phase: :scheduled}, {:handlerError, true}) == []
    end

    # What the Model actually reaches: every phase, and no state it can get stuck in.
    test "the product machine reaches all 6 phases" do
      reached = NexusProduct.table() |> Table.reachable() |> Enum.map(& &1.phase) |> Enum.sort()
      assert reached == Enum.sort(ProductPhase.values())
      assert Table.stuck(NexusProduct.table()) == nil
    end

    # Eight phases, three attempt counts and three deadlines, and the four phases it ends on.
    test "the protocol machine has 192 states, 96 ends and 23 action classes" do
      table = NexusProtocol.table()
      assert length(table.states) == 8 * 3 * 2 * 2 * 2
      assert length(table.ends) == 4 * 3 * 8
      assert action_classes(table) == 8 + 6 + 3 + 1 + 1 + 4
      assert table.starts == [at(Nexus.ProtocolState, :unscheduled)]
    end

    # The refinement: every protocol row is a product row between its mapped states, or a stutter.
    test "the refinement passes" do
      refinement = NexusProtocol.refinement()
      assert refinement.rejected == nil
      assert length(refinement.rows) == length(NexusProtocol.table().rows)

      scheduled = at(Nexus.ProtocolState, :scheduled)
      # A reply the product sees is that reply's row; a retry it cannot see is a stutter.
      assert {:matches, %Table.Row{class: {:handlerReply, [:async]}}} = verdict(refinement, scheduled, {:handlerReply, [:async]})
      assert verdict(refinement, scheduled, {:handlerReply, [{:handlerError, true}]}) == :stutter

      # A deadline firing is the product's one timer, whichever deadline it was.
      started = %{at(Nexus.ProtocolState, :started) | startToClose: :expires}
      assert {:matches, %Table.Row{class: {:timeout, []}}} = verdict(refinement, started, {:startToClose, []})
    end

    test "every find query finds its claim on its path" do
      for query <- [:syncCompletion, :asyncCompletion, :asyncFailure, :handlerError, :retry, :scheduleToStartTimeout, :startToCloseTimeout] do
        assert_query(Nexus, query, :found)
      end
    end

    # The product claim, read through the map, over every trace of the asynchronous path.
    test "terminalHolds verifies" do
      assert %Search.Result{fired: fired} = assert_query(Nexus, :terminalHolds, :verified_within_limits)
      assert fired > 0
    end

    # Verified over the composition: the one reply comes before the stop, and the claim fired on
    # it, so the verify was exercised.
    test "stoppedWorkerRepliesNothing verifies and is exercised" do
      assert %Search.Result{fired: 1} = assert_query(Nexus, :stoppedWorkerRepliesNothing, :verified_within_limits)
      assert length(Nexus.__umpire__({:table, :nexusCaller}).states) == 2 * length(NexusProtocol.table().states)
    end

    test "the compiled steps agree with the IR on every row" do
      assert_agrees(Nexus, :nexusProduct)
      assert_agrees(Nexus, :nexusProtocol)
    end
  end

  describe "standalone activity" do
    alias Activity.{ActivityProduct, ActivityProtocol, ProtocolState}

    # Nine phases the caller can read, and the five the design ends on.
    test "the product machine has 9 states and 5 ends" do
      assert length(ActivityProduct.table().states) == 9
      assert length(ActivityProduct.table().ends) == 5
    end

    # Twelve phases, three attempt counts and three deadlines.
    test "the protocol machine has 288 states and 120 ends" do
      assert length(ActivityProtocol.table().states) == 12 * 3 * 8
      assert length(ActivityProtocol.table().ends) == 5 * 3 * 8
    end

    # A cancel response with no cancel requested is not enabled.
    test "attemptResult(:canceled) from started is not enabled" do
      assert ActivityProtocol.attempt_result(at(ProtocolState, :started), :canceled) == []
    end

    # The visible retry: the row records the status Describe reads as well as the count, which is
    # what the strict rule needs to match it to the product's retry row.
    test "a retryable failure from started backs off and records SCHEDULED and the count" do
      started = %{at(ProtocolState, :started) | attempts: 1}

      assert ActivityProtocol.attempt_result(started, {:failed, true}) == [
               %Step{outcome: :accepted, state: %{started | phase: :backingOff}, facts: [:statusScheduled, :attemptCount]}
             ]
    end

    # The refinement: a retryable failure is visible (started -> backingOff reads as started ->
    # scheduled), a pause request is a stutter (pauseRequested reads as started), and a retryable
    # failure under a requested pause reads as the product's pause.
    test "the refinement passes" do
      refinement = ActivityProtocol.refinement()
      assert refinement.rejected == nil
      assert refinement.rule == :strict

      started = %{at(ProtocolState, :started) | attempts: 1}
      pause_requested = %{started | phase: :pauseRequested}
      backing_off = %{started | phase: :backingOff}

      assert {:matches, %Table.Row{class: {:attemptResult, [{:failed, true}]}}} =
               verdict(refinement, started, {:attemptResult, [{:failed, true}]})

      assert verdict(refinement, started, {:control, [:pause]}) == :stutter
      assert verdict(refinement, pause_requested, {:control, [:unpause]}) == :stutter
      assert {:matches, %Table.Row{class: {:control, [:pause]}}} = verdict(refinement, pause_requested, {:attemptResult, [{:failed, true}]})
      assert {:matches, %Table.Row{class: {:control, [:pause]}}} = verdict(refinement, backing_off, {:control, [:pause]})
    end

    # The second revision note, pinned: without :statusScheduled on the retry row the strict rule
    # rejects it, and the mapped-states rule does not notice.
    test "the strict rule is what needs the extra fact" do
      table = ActivityProtocol.table()
      product = ActivityProduct.table()
      mapped = Map.new(table.states, &{&1, ActivityProtocol.product_of(&1)})
      evidence = {ActivityProtocol.__umpire_machine__().evidence, ActivityProduct.__umpire_machine__().evidence}

      stripped = %{
        table
        | rows:
            Enum.map(table.rows, fn
              %Table.Row{from: %{phase: :started}, class: {:attemptResult, [{:failed, true}]}} = row -> %{row | facts: [:attemptCount]}
              row -> row
            end)
      }

      assert %Refinement.Result{rejected: {%Table.Row{class: {:attemptResult, [{:failed, true}]}}, _, _}} =
               Refinement.check(stripped, product, mapped, :strict, evidence)

      assert %Refinement.Result{rejected: nil} = Refinement.check(stripped, product, mapped, :mapped_states, evidence)
    end

    test "every find query finds its claim on its path" do
      for query <- [:completion, :nonRetryableFailure, :retry, :cancel, :terminate, :pauseResume, :scheduleToStartTimeout, :startToCloseTimeout] do
        assert_query(Activity, query, :found)
      end
    end

    # The two product claims, read through the map, over every trace of their paths.
    test "terminalHolds and pauseHolds verify" do
      assert_query(Activity, :terminalHolds, :verified_within_limits)
      assert_query(Activity, :pauseHolds, :verified_within_limits)
    end

    # The third revision note, pinned. The composition verify is exercised: the one attempt starts
    # while the worker polls and the claim fires on it.
    test "stoppedWorkerStartsNothing verifies and its claim fires" do
      assert %Search.Result{fired: 1, paths: 1} = assert_query(Activity, :stoppedWorkerStartsNothing, :verified_within_limits)
    end

    # And the path it replaced would now be an error rather than a pass: it never performs
    # attemptStart, so the claim never fires.
    test "the old stoppedBeforeDispatch path is vacuous" do
      model = Activity.__umpire__(:ir)
      [start, _attempt | _] = Enum.find(model.scenarios, &(&1.name == :stoppedBeforeRetry)).actions
      worker_stop = %IR.ClassRef{action: :workerStop}
      timer = %IR.ClassRef{member: :activity, action: :scheduleToStart}

      old = %IR.Scenario{name: :stoppedBeforeDispatch, model: :standaloneActivity, starts: [activity: :unstarted], actions: [start, worker_stop, timer]}
      query = %IR.Query{name: :stoppedBeforeDispatchQuery, kind: :verify, property: :startedByPollingWorker, scenario: old.name, limits: :six}
      model = %{model | scenarios: [old | model.scenarios], queries: [query | model.queries]}

      assert %Search.Result{outcome: :vacuous, fired: 0} = Search.run(model, query)
    end

    test "the compiled steps agree with the IR on every row" do
      assert_agrees(Activity, :activityProduct)
      assert_agrees(Activity, :activityProtocol)
    end
  end
end
