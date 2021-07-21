package testdata

import (
	"database/sql"
)

type UserSRMRow struct {
	User
}

func (row *UserSRMRow) bind(rows *sql.Rows, columns []string) error {
	var (
		dest []interface{}
		counterOfage int
		counterOfcreatedat int
		counterOfemail int
		counterOfid int
		counterOfname int
	)
	for _, name := range columns {
		switch name {
		case "age":
			switch counterOfage {
			case 0:
				dest = append(dest, &row.User.Age)
				counterOfage++
				continue
			}
			counterOfage++
		case "createdat":
			switch counterOfcreatedat {
			case 0:
				dest = append(dest, &row.User.CreatedAt)
				counterOfcreatedat++
				continue
			}
			counterOfcreatedat++
		case "email":
			switch counterOfemail {
			case 0:
				dest = append(dest, &row.User.Emailer.Email)
				counterOfemail++
				continue
			}
			counterOfemail++
		case "id":
			switch counterOfid {
			case 0:
				dest = append(dest, &row.User.ID)
				counterOfid++
				continue
			case 1:
				dest = append(dest, &row.User.Emailer.ID)
				counterOfid++
				continue
			}
			counterOfid++
		case "name":
			switch counterOfname {
			case 0:
				dest = append(dest, &row.User.Name)
				counterOfname++
				continue
			}
			counterOfname++
		}
		var i interface{}
		dest = append(dest, &i)
	}
	return rows.Scan(dest...)
}

type UserSRM []UserSRMRow
func (srm *UserSRM) Bind(rows *sql.Rows, err error) error {
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		var srmRow UserSRMRow
		err := srmRow.bind(rows, columns)
		if err != nil {
			return err
		}
		*srm = append(*srm, srmRow)
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}
