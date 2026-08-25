package migrations

import "embed"

//go:embed db/migrations/*.sql
var Embedded embed.FS
