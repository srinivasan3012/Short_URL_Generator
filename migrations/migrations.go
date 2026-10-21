package migrations

import _ "embed"

//go:embed 000001_create_urls_table.up.sql
var Migration001Up string

//go:embed 000001_create_urls_table.down.sql
var Migration001Down string
