// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"

	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/stretchr/testify/require"
)

func TestBindingAccessor(t *testing.T) {
	type NewContextArgs struct {
		Args, Res []reflect.Value
		Req       types.RequestReader
		Values    interface{}
	}

	db := sql.OpenDB(fakeSQLDriver{})

	type TestCase struct {
		Expr          string
		ExpectedValue interface{}
		ExpectedError bool
	}

	for _, tc := range []struct {
		Name           string
		Capabilities   []string
		NewContextArgs NewContextArgs
		TestCases      []TestCase
	}{
		{
			Name:         "SQL",
			Capabilities: []string{"sql", "rule", "func"},
			NewContextArgs: NewContextArgs{
				Args: []reflect.Value{
					reflect.ValueOf(&db),
				},
				Values: map[string]interface{}{
					"dialects": map[string]interface{}{
						"mysql":  []interface{}{"mypkg", reflect.TypeOf(fakeSQLDriver{}).PkgPath()},
						"mysql2": []interface{}{"mypkg2"},
					},

					"dialects2": map[string]interface{}{
						"mysql":  []interface{}{"mypkg"},
						"mysql2": []interface{}{"mypkg2"},
					},

					"dialects_wrong_type": map[string][]string{
						"mysql":  {"mypkg"},
						"mysql2": {"mypkg2"},
					},
				},
			},
			TestCases: []TestCase{
				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], #.Rule.Data.Values['dialects'])",
					ExpectedValue: "mysql",
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], #.Rule.Data.Values['dialects2'])",
					ExpectedValue: nil,
					ExpectedError: true,
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], #.Rule.Data.Values['oops'])",
					ExpectedValue: nil,
					ExpectedError: true,
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], #.Rule.Data.Oops)",
					ExpectedValue: nil,
					ExpectedError: true,
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], nil)",
					ExpectedValue: nil,
					ExpectedError: true,
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[0], #.Rule.Data.Values['dialects_wrong_type'])",
					ExpectedValue: nil,
					ExpectedError: true,
				},

				{
					Expr:          "#.SQL.Dialect(#.Func.Args[1], #.Rule.Data.Values['dialects'])",
					ExpectedValue: nil,
					ExpectedError: true,
				},
			},
		},

		{
			Name:         "Array Library",
			Capabilities: []string{"lib", "rule"},
			NewContextArgs: NewContextArgs{
				Values: struct {
					StringSlice, EmptyStringSlice []string
					StringValue                   string

					IntSlice, EmptyIntSlice []int
					IntValue                int
					EmptyInterfaceSlice     []interface{}
				}{
					StringSlice:      []string{"b", "c", "d"},
					EmptyStringSlice: []string{},
					StringValue:      "a",

					IntSlice:      []int{2, 3, 4},
					EmptyIntSlice: []int{},
					IntValue:      1,

					EmptyInterfaceSlice: []interface{}{},
				},
			},
			TestCases: []TestCase{
				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.StringSlice, #.Rule.Data.Values.StringValue)",
					ExpectedValue: []string{"a", "b", "c", "d"},
				},
				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.StringSlice, 'a')",
					ExpectedValue: []string{"a", "b", "c", "d"},
				},
				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.EmptyStringSlice, 'a')",
					ExpectedValue: []string{"a"},
				},
				{
					Expr:          "#.Lib.Array.Prepend(nil, 'a')",
					ExpectedValue: []string{"a"},
				},
				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.StringSlice, #.Rule.Data.Values.IntValue)",
					ExpectedError: true,
				},

				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.IntSlice, #.Rule.Data.Values.IntValue)",
					ExpectedValue: []int{1, 2, 3, 4},
				},
				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.EmptyIntSlice, #.Rule.Data.Values.IntValue)",
					ExpectedValue: []int{1},
				},
				{
					Expr:          "#.Lib.Array.Prepend(nil, #.Rule.Data.Values.IntValue)",
					ExpectedValue: []int{1},
				},

				{
					Expr:          "#.Lib.Array.Prepend(nil, #.Rule.Data.Values.IntSlice)",
					ExpectedValue: [][]int{{2, 3, 4}},
				},

				{
					Expr:          "#.Lib.Array.Prepend(#.Rule.Data.Values.EmptyInterfaceSlice, #.Rule.Data.Values.IntValue)",
					ExpectedValue: []interface{}{1},
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			ctx, err := callback.NewCallbackBindingAccessorContext(tc.Capabilities, tc.NewContextArgs.Args, tc.NewContextArgs.Res, tc.NewContextArgs.Req, tc.NewContextArgs.Values)
			require.NoError(t, err)

			for _, tc := range tc.TestCases {
				tc := tc
				t.Run("", func(t *testing.T) {
					ba, err := bindingaccessor.Compile(tc.Expr)
					require.NoError(t, err)

					v, err := ba(ctx)
					if tc.ExpectedError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
					require.Equal(t, tc.ExpectedValue, v)
				})
			}
		})
	}
}

type fakeSQLDriver struct{}

func (f fakeSQLDriver) Open(string) (driver.Conn, error)             { return nil, nil }
func (f fakeSQLDriver) Connect(context.Context) (driver.Conn, error) { return nil, nil }
func (f fakeSQLDriver) Driver() driver.Driver                        { return f }
