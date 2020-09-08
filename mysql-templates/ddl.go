package mysql_templates

import (
	"strings"
)

type sql struct {
	s string
}

func fromStr(s string) sql {
	return sql{s: s}
}

func (s sql) apply(isLower bool) string {
	if isLower {
		return strings.ToLower(s.s)
	}

	return s.s
}

func CreateTableStm(isLower bool) string {
	return fromStr("CREATE TABLE %s (\n%s\n);").apply(isLower)
}

func DropTableStm(isLower bool) string {
	return fromStr("DROP TABLE IF EXISTS %s;").apply(isLower)
}

func RenameTableStm(isLower bool) string {
	return fromStr("ALTER TABLE %s RENAME TO %s;").apply(isLower)
}

func AlterTableAddColumnFirstStm(isLower bool) string {
	return fromStr("ALTER TABLE %s ADD COLUMN %s FIRST;").apply(isLower)
}

func AlterTableAddColumnAfterStm(isLower bool) string {
	return fromStr("ALTER TABLE %s ADD COLUMN %s AFTER %s;").apply(isLower)
}

func AlterTableDropColumnStm(isLower bool) string {
	return fromStr("ALTER TABLE %s DROP COLUMN %s;").apply(isLower)
}

func AlterTableModifyStm(isLower bool) string {
	return fromStr("ALTER TABLE %s MODIFY COLUMN %s;").apply(isLower)
}

func AlterTableRenameStm(isLower bool) string {
	return fromStr("ALTER TABLE %s RENAME COLUMN %s TO %s;").apply(isLower)
}

func CreateIndexStm(isLower bool) string {
	return fromStr("CREATE INDEX %s ON %s(%s);").apply(isLower)
}

func CreateUniqueIndexStm(isLower bool) string {
	return fromStr("CREATE UNIQUE INDEX %s ON %s(%s);").apply(isLower)
}

func DropIndexStm(isLower bool) string {
	return fromStr("DROP INDEX %s ON %s;").apply(isLower)
}
