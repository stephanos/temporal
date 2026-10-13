package framework.`export`

@main def lintConventions(jars: String, classpathFiles: String*): Unit =
  framework.lint.lintConventions(jars, classpathFiles, Plugin.installed)
