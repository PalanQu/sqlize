package mysql_parser

import (
	"strings"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
)

var (
	isLower bool
)

type Migration struct {
	currentTable string
	Tables       []Table
	tableIndexes map[string]int
}

func NewMigration(isLowercase bool) Migration {
	isLower = isLowercase

	return Migration{
		Tables:       []Table{},
		tableIndexes: map[string]int{},
	}
}

func (m *Migration) Parser(sql string) error {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return err
	}

	for _, n := range stmtNodes {
		switch n.(type) {
		case ast.DDLNode:
			n.Accept(m)
			break
		}
	}
	return nil
}

func (m *Migration) Enter(in ast.Node) (ast.Node, bool) {
	// drop Table
	if tb, ok := in.(*ast.DropTableStmt); ok {
		for i := range tb.Tables {
			m.removeTable(tb.Tables[i].Name.O)
		}
	}

	// alter Table
	if alter, ok := in.(*ast.AlterTableStmt); ok {
		for i := range alter.Specs {
			switch alter.Specs[i].Tp {
			case ast.AlterTableAddColumns:
				break
			case ast.AlterTableDropColumn:
				m.removeColumn(alter.Table.Name.O, alter.Specs[i].OldColumnName.Name.O)
				break
			case ast.AlterTableModifyColumn:
				// TODO
				break
			case ast.AlterTableRenameColumn:
				// TODO
				break
			case ast.AlterTableRenameTable:
				// TODO
				break
			case ast.AlterTableRenameIndex:
				// TODO
				break
			}
		}
	}

	// drop Index
	if idx, ok := in.(*ast.DropIndexStmt); ok {
		m.removeIndex(idx.Table.Name.O, idx.IndexName)
	}

	// alter Table
	if tb, ok := in.(*ast.TableName); ok {
		m.using(tb.Name.O)
	}

	// create Table
	if tab, ok := in.(*ast.CreateTableStmt); ok {
		m.using(tab.Table.Name.O)

		tb := NewTable(tab.Table.Name.O)
		for i := range tab.Constraints {
			cols := make([]string, len(tab.Constraints[i].Keys))
			for j, key := range tab.Constraints[i].Keys {
				cols[j] = key.Column.Name.O
			}

			switch tab.Constraints[i].Tp {
			case ast.ConstraintPrimaryKey:
				tb.addColumn(Column{
					Node: Node{Name: cols[0], Action: MigrateAddAction},
					Typ:  nil,
					Options: []*ast.ColumnOption{
						{
							Tp: ast.ColumnOptionPrimaryKey,
						},
					},
				})
				break
			case ast.ConstraintKey, ast.ConstraintIndex:
				tb.addIndex(Index{
					Node: Node{
						Name:   tab.Constraints[i].Name,
						Action: MigrateAddAction,
					},
					Typ:     ast.IndexKeyTypeNone,
					Columns: cols,
				})
				break
			case ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
				tb.addIndex(Index{
					Node: Node{
						Name:   tab.Constraints[i].Name,
						Action: MigrateAddAction,
					},
					Typ:     ast.IndexKeyTypeUnique,
					Columns: cols,
				})
				break
			}
		}

		m.addTable(*tb)
	}

	// define Column
	if def, ok := in.(*ast.ColumnDef); ok {
		column := Column{
			Node:    Node{Name: def.Name.Name.O, Action: MigrateAddAction},
			Typ:     def.Tp,
			Options: def.Options,
		}
		m.addColumn("", column)
	}

	// create Index
	if idx, ok := in.(*ast.CreateIndexStmt); ok {
		cols := make([]string, len(idx.IndexPartSpecifications))
		for i := range idx.IndexPartSpecifications {
			cols[i] = idx.IndexPartSpecifications[i].Column.Name.O
		}

		m.addIndex(idx.Table.Name.O, Index{
			Node: Node{
				Name:   idx.IndexName,
				Action: MigrateAddAction,
			},
			Typ:     idx.KeyType,
			Columns: cols,
		})
	}

	return in, false
}

func (m *Migration) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (m *Migration) using(tbName string) {
	if tbName != "" {
		m.currentTable = tbName
	}
}

func (m *Migration) addTable(tb Table) {
	id := m.getIndexTable(tb.Name)
	if id == -1 {
		m.Tables = append(m.Tables, tb)
		m.tableIndexes[tb.Name] = len(m.Tables) - 1
		return
	}

	m.Tables[id] = tb
}

func (m *Migration) removeTable(tbName string) {
	id := m.getIndexTable(tbName)
	if id == -1 {
		tb := NewTable(tbName)
		tb.Action = MigrateRemoveAction
		m.Tables = append(m.Tables, *tb)
		m.tableIndexes[tb.Name] = len(m.Tables) - 1
		return
	}

	if m.Tables[id].Action == MigrateAddAction {
		m.Tables[id].Action = MigrateNoAction
	} else {
		m.Tables[id].Action = MigrateRemoveAction
	}
}

func (m *Migration) addColumn(tbName string, col Column) {
	if tbName == "" {
		tbName = m.currentTable
	}

	id := m.getIndexTable(tbName)
	if id == -1 {
		tb := NewTable(tbName)
		tb.Action = MigrateNoAction
		m.addTable(*tb)
		id = len(m.Tables) - 1
	}

	m.Tables[id].addColumn(col)
}

func (m *Migration) removeColumn(tbName string, colName string) {
	if tbName == "" {
		tbName = m.currentTable
	}

	id := m.getIndexTable(tbName)
	if id == -1 {
		m.addTable(*NewTable(tbName))
		id = len(m.Tables) - 1
	}

	m.Tables[id].removeColumn(colName)
}

func (m *Migration) addIndex(tbName string, idx Index) {
	if tbName == "" {
		tbName = m.currentTable
	}

	id := m.getIndexTable(tbName)
	if id == -1 {
		m.addTable(*NewTable(tbName))
		id = len(m.Tables) - 1
	}

	m.Tables[id].addIndex(idx)
}

func (m *Migration) removeIndex(tbName string, idxName string) {
	if tbName == "" {
		tbName = m.currentTable
	}

	id := m.getIndexTable(tbName)
	if id == -1 {
		m.addTable(*NewTable(tbName))
		id = len(m.Tables) - 1
	}

	m.Tables[id].removeIndex(idxName)
}

func (m Migration) getIndexTable(tableName string) int {
	if v, ok := m.tableIndexes[tableName]; ok {
		return v
	}

	return -1
}

func (m *Migration) Diff(old Migration) {
	for i := range m.Tables {
		if j := old.getIndexTable(m.Tables[i].Name); j >= 0 {
			m.Tables[i].Diff(old.Tables[j])
			m.Tables[i].Action = MigrateNoAction
		}
	}

	for j := range old.Tables {
		if i := m.getIndexTable(old.Tables[j].Name); i == -1 {
			old.Tables[j].Action = MigrateRemoveAction
			m.addTable(old.Tables[j])
		}
	}
}

func (m Migration) MigrationUp() string {
	strTables := make([]string, 0)
	for i := range m.Tables {
		strTb := append([]string{}, strings.Join(m.Tables[i].migrationColumnUp(), "\n"), strings.Join(m.Tables[i].migrationIndexUp(), "\n"))
		strTables = append(strTables, strings.Join(strTb, "\n"))
	}

	return strings.Join(strTables, "\n\n")
}

func (m Migration) MigrationDown() string {
	strTables := make([]string, 0)
	for i := range m.Tables {
		strTb := append([]string{}, strings.Join(m.Tables[i].migrationColumnDown(), "\n"), strings.Join(m.Tables[i].migrationIndexDown(), "\n"))
		strTables = append(strTables, strings.Join(strTb, "\n"))
	}

	return strings.Join(strTables, "\n\n")
}
