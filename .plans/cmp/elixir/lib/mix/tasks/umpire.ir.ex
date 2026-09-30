defmodule Mix.Tasks.Umpire.Ir do
  @shortdoc "Writes each Model's IR as canonical JSON"

  @moduledoc """
  Writes `priv/umpire/<Model>.ir.json` for every compiled module that is an Umpire Model.

      mix umpire.ir
      mix umpire.ir --check    # fail if a file on disk differs, for CI

  A mix task rather than `@after_compile`: a compile callback that writes files would run on
  every recompilation, race the parallel compiler, and write outside what Mix tracks. The task
  reads the IR the Models already carry (`Model.__umpire__(:ir)`), so it adds no compile work.
  """

  use Mix.Task

  @requirements ["app.config"]

  @impl Mix.Task
  def run(args) do
    {opts, _, _} = OptionParser.parse(args, strict: [check: :boolean])
    dir = Path.join(File.cwd!(), "priv/umpire")
    File.mkdir_p!(dir)

    stale =
      for module <- models(), reduce: [] do
        stale ->
          path = Path.join(dir, "#{inspect(module)}.ir.json")
          json = IO.iodata_to_binary(Umpire.IR.to_json(module.__umpire__(:ir)))

          cond do
            !opts[:check] ->
              File.write!(path, json)
              stale

            File.read(path) == {:ok, json} ->
              stale

            true ->
              [path | stale]
          end
      end

    if stale != [], do: Mix.raise("IR out of date, run mix umpire.ir: #{Enum.join(stale, ", ")}")
  end

  defp models do
    {:ok, modules} = :application.get_key(Mix.Project.config()[:app], :modules)
    Enum.filter(modules, &(Code.ensure_loaded?(&1) and function_exported?(&1, :__umpire__, 1)))
  end
end
