package models

// AppVersion is the version shown in the main menu. Release builds can override
// it with -ldflags "-X mcget/models.AppVersion=<version>".
var AppVersion = "1.0.0"
