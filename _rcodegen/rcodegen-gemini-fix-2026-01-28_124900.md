--- pkg/settings/settings.go
+++ pkg/settings/settings.go
@@ -167,11 +167,11 @@
 
 // LoadWithFallback tries to load settings, falling back to defaults if not found
 // Returns the settings (possibly with defaults) and whether the config file existed
-func LoadWithFallback() (*Settings, bool) {
+func LoadWithFallback() (*Settings, bool, error) {
 	settings, err := Load()
 	if err != nil {
- 		return GetDefaultSettings(), false
+ 		return GetDefaultSettings(), false, nil
 	}
 	// Fill in any missing defaults
 	if settings.Defaults.Codex.Model == "" {
@@ -194,17 +194,16 @@
 	}
 	// Check for reserved task name overrides before merging
 	if err := ValidateNoReservedTaskOverrides(settings.Tasks); err != nil {
-		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", yellow, reset, err)
-		os.Exit(1)
+ 		return nil, false, err
 	}
 	// Merge default tasks - custom user tasks with non-reserved names are allowed
 	if settings.Tasks == nil {
 		settings.Tasks = make(map[string]TaskDef)
 	}
 	for name, task := range GetDefaultTasks() {
 		settings.Tasks[name] = task // Always use built-in defaults for reserved names
 	}
- 	return settings, true
+ 	return settings, true, nil
 }
 
 // ToTaskConfig converts Settings to the legacy TaskConfig format
