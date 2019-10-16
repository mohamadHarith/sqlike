package examples

import (
	"testing"

	"github.com/si3nloong/sqlike/sqlike"
	"github.com/si3nloong/sqlike/sqlike/options"
	"github.com/stretchr/testify/require"
)

type rsqlStruct struct {
	ID     int64 `sqlike:",primary_key"`
	Status Enum  `sqlike:",enum=SUCCESS|FAILED|UNKNOWN"`
}

type queryStruct struct {
	ID     int64  `rsql:"id,select,filter,sort"`
	Status string `rsql:",filter,sort"`
}

// RSQLExamples :
func RSQLExamples(t *testing.T, db *sqlike.Database) {
	var (
		// parser *rsql.Parser
		// params *rsql.Params
		err error
	)

	table := db.Table("rsql_struct")

	{
		var src ***rsqlStruct
		err = table.UnsafeMigrate(src)
		require.NoError(t, err)

		err = table.Truncate()
		require.NoError(t, err)
	}

	{
		data := []rsqlStruct{
			rsqlStruct{ID: 1, Status: Failed},
			rsqlStruct{ID: 2},
			rsqlStruct{ID: 3},
			rsqlStruct{ID: 4, Status: Failed},
		}
		_, err = table.Insert(&data, options.Insert().SetDebug(true))
		require.NoError(t, err)
	}

	// var src **queryStruct
	// parser = rsql.MustNewParser(src)
	// require.NotNil(t, parser)

	// query := `$select=&$filter=(id==value)&$sort=&$limit=100`

	// {

	// 	params, err = parser.ParseQuery(query)
	// 	require.NoError(t, err)
	// 	require.NotNil(t, params)

	// 	_, err = table.Find(actions.Find().
	// 		Where(
	// 			params.Filters,
	// 			expr.Equal("Status", Success),
	// 		), options.Find().SetDebug(true))
	// 	require.NoError(t, err)

	// 	log.Println(parser)
	// 	log.Println(params.Filters)

	// }
}