package element

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	ptypes "github.com/auxten/postgresql-parser/pkg/sql/types"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"github.com/pingcap/parser/types"
	"github.com/sunary/sqlize/utils"
)

const (
	// UppercaseRestoreFlag ...
	UppercaseRestoreFlag = format.RestoreStringSingleQuotes | format.RestoreKeyWordUppercase | format.RestoreNameUppercase | format.RestoreNameBackQuotes
	// LowerRestoreFlag ...
	LowerRestoreFlag = format.RestoreStringSingleQuotes | format.RestoreKeyWordLowercase | format.RestoreNameLowercase | format.RestoreNameBackQuotes
)

// Column ...
type Column struct {
	Node
	Typ     *types.FieldType
	PgTyp   *ptypes.InternalType
	Options []*ast.ColumnOption
	Comment string
}

// GetType ...
func (c Column) GetType() byte {
	if c.Typ != nil {
		return c.Typ.Tp
	}

	return 0
}

// HasDefaultValue ...
func (c Column) HasDefaultValue() bool {
	for _, opt := range c.Options {
		if opt.Tp == ast.ColumnOptionDefaultValue {
			return true
		}
	}

	return false
}

func (c Column) hashValue() string {
	strHash := utils.EscapeSqlName(sql.IsPostgres, c.Name)
	strHash += c.hashDefinition()
	hash := md5.Sum([]byte(strHash))
	return hex.EncodeToString(hash[:])
}

func (c Column) migrationUp(tbName, after string, ident int) []string {
	switch c.Action {
	case MigrateNoAction:
		return nil

	case MigrateAddAction:
		strSql := utils.EscapeSqlName(sql.IsPostgres, c.Name)

		if ident > len(c.Name) {
			strSql += strings.Repeat(" ", ident-len(c.Name))
		}

		strSql += c.definition()

		if ident < 0 {
			if after != "" {
				return []string{fmt.Sprintf(sql.AlterTableAddColumnAfterStm(), utils.EscapeSqlName(sql.IsPostgres, tbName), strSql, utils.EscapeSqlName(sql.IsPostgres, after))}
			}
			return []string{fmt.Sprintf(sql.AlterTableAddColumnFirstStm(), utils.EscapeSqlName(sql.IsPostgres, tbName), strSql)}
		}

		return []string{strSql}

	case MigrateRemoveAction:
		return []string{fmt.Sprintf(sql.AlterTableDropColumnStm(), utils.EscapeSqlName(sql.IsPostgres, tbName), utils.EscapeSqlName(sql.IsPostgres, c.Name))}

	case MigrateModifyAction:
		return []string{fmt.Sprintf(sql.AlterTableModifyColumnStm(), utils.EscapeSqlName(sql.IsPostgres, tbName), utils.EscapeSqlName(sql.IsPostgres, c.Name)+c.definition())}

	case MigrateRenameAction:
		return []string{fmt.Sprintf(sql.AlterTableRenameColumnStm(), utils.EscapeSqlName(sql.IsPostgres, tbName), utils.EscapeSqlName(sql.IsPostgres, c.OldName), utils.EscapeSqlName(sql.IsPostgres, c.Name))}

	default:
		return nil
	}
}

func (c Column) migrationDown(tbName, after string) []string {
	switch c.Action {
	case MigrateNoAction:
		return nil

	case MigrateAddAction:
		c.Action = MigrateRemoveAction

	case MigrateRemoveAction:
		c.Action = MigrateAddAction

	case MigrateModifyAction:
		return nil

	case MigrateRenameAction:
		c.Name, c.OldName = c.OldName, c.Name

	default:
		return nil
	}

	return c.migrationUp(tbName, after, -1)
}

func (c Column) definition() string {
	strSql := ""
	if !sql.IsPostgres && c.Typ != nil {
		strSql += " " + c.Typ.String()
	} else if sql.IsPostgres && c.PgTyp != nil {
		strSql += " " + sql.FamilyName(int32(c.PgTyp.Family))
	}

	for _, opt := range c.Options {
		b := bytes.NewBufferString("")
		var ctx *format.RestoreCtx
		if sql.IsLower {
			ctx = format.NewRestoreCtx(LowerRestoreFlag, b)
		} else {
			ctx = format.NewRestoreCtx(UppercaseRestoreFlag, b)
		}

		if sql.IsPostgres && opt.Tp == ast.ColumnOptionDefaultValue {
			strSql += " " + b.String()
			continue
		}

		_ = opt.Restore(ctx)
		strSql += " " + b.String()
	}

	return strSql
}

func (c Column) hashDefinition() string {
	strSql := ""
	if !sql.IsPostgres && c.Typ != nil {
		strSql += " " + c.Typ.String()
	} else if sql.IsPostgres && c.PgTyp != nil {
		strSql += " " + sql.FamilyName(int32(c.PgTyp.Family))
	}

	return strSql
}
