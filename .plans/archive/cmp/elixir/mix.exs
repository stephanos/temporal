defmodule Umpire.MixProject do
  use Mix.Project

  # The three Model files stay at the top of this directory, beside the other languages'
  # samples, and are compiled from there; in a real repository they would live under lib/.
  @models ["worker.ex", "nexus_caller.ex", "standalone_activity.ex"]

  def project do
    [
      app: :umpire_models,
      version: "0.1.0",
      elixir: "~> 1.20",
      elixirc_paths: ["lib" | @models],
      deps: []
    ]
  end

  def application, do: [extra_applications: [:logger]]
end
