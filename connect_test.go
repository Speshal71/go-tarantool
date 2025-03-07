package tarantool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	box, err := NewBox("", nil)
	require.NoError(err)
	defer box.Close()

	conn, err := Connect(box.Addr(), nil)
	require.NoError(err)
	defer conn.Close()

	assert.NotEqual(conn.greeting.Version, 0)
}

func TestMapIndexDescription(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	config := `
	local s = box.schema.space.create('tester', {id = 42})
	s:create_index('tester_id', {
		parts = {
        	{field = 1, type = 'number', is_nullable = false},
    	},
	})
	local t = s:insert({1})
	`
	box, err := NewBox(config, nil)
	require.NoError(err)
	defer box.Close()

	conn, err := Connect(box.Addr(), nil)
	require.NoError(err)
	defer conn.Close()

	pkFields, ok := conn.GetPrimaryKeyFields("tester")
	require.True(ok)
	assert.ElementsMatch(pkFields, []int{0})
}

func TestDefaultSpace(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	config := `
	local s = box.schema.space.create('tester', {id = 42})
	s:create_index('tester_id', {
		type = 'hash',
		parts = {1, 'NUM'}
	})
	local t = s:insert({1})
	`
	box, err := NewBox(config, nil)
	require.NoError(err)
	defer box.Close()

	conn, err := Connect(box.Addr(), &Options{
		DefaultSpace: "tester",
	})
	require.NoError(err)
	defer conn.Close()

	tuples, err := conn.Execute(&Select{
		Key:   1,
		Index: "tester_id",
	})
	require.NoError(err)
	assert.Equal([][]interface{}{{int64(1)}}, tuples)
}

func TestConnectOptionsDSN(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		uri    string
		user   string
		pass   string
		scheme string
		host   string
		space  string
		err    error
	}{
		// for backward compatibility
		{"unix://127.0.0.1", "", "", "tcp", "127.0.0.1", "", nil},
		// scheme, host, user, pass
		{"tcp://127.0.0.1", "", "", "tcp", "127.0.0.1", "", nil},
		{"//127.0.0.1", "", "", "tcp", "127.0.0.1", "", nil},
		{"127.0.0.1", "", "", "tcp", "127.0.0.1", "", nil},
		{"tcp://user:pass@127.0.0.1:8000", "user", "pass", "tcp", "127.0.0.1:8000", "", nil},
		{"127.0.0.1:8000", "", "", "tcp", "127.0.0.1:8000", "", nil},
		{"user:pass@127.0.0.1:8000", "user", "pass", "tcp", "127.0.0.1:8000", "", nil},
		// path (defaultSpace)
		{"127.0.0.1/", "", "", "tcp", "127.0.0.1", "", ErrEmptyDefaultSpace},
		{"127.0.0.1/tester", "", "", "tcp", "127.0.0.1", "tester", nil},
		// no errors due to disabled checks
		{"127.0.0.1/tester/1", "", "", "tcp", "127.0.0.1", "tester/1", nil},
		{"127.0.0.1/tester%20two", "", "", "tcp", "127.0.0.1", "tester two", nil},
		{"127.0.0.1/tester%2Ctwo", "", "", "tcp", "127.0.0.1", "tester,two", nil},
	}
	for tc, item := range tt {
		dsn, opts, err := parseOptions(item.uri, Options{})
		assert.Equal(item.err, err, "case %v (err)", tc+1)
		if err != nil {
			continue
		}
		assert.Equal(item.scheme, dsn.Scheme, "case %v (scheme)", tc+1)
		assert.Equal(item.host, dsn.Host, "case %v (host)", tc+1)
		assert.Equal(item.user, opts.User, "case %v (user)", tc+1)
		assert.Equal(item.pass, opts.Password, "case %v (password)", tc+1)
		assert.Equal(item.space, opts.DefaultSpace, "case %v (space)", tc+1)
	}

}

// TestConnectionWithDefaultResultUnmarshalMode tests that
// overwriting the result' unmarshal mode doesn't interferer with internal queries
// like auth and schema pulling.
func TestConnectionWithDefaultResultUnmarshalMode(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	config := `
	local s = box.schema.space.create('tester', {id = 42})
	s:create_index('tester_id', {
		type = 'hash',
		parts = {1, 'NUM'}
	})
	local t = s:insert({33, 45})

	box.schema.user.create("tester", {password = "12345678"})
	box.schema.user.grant('tester', 'read', 'space', 'tester')
	`

	box, err := NewBox(config, nil)
	require.NoError(err)
	defer box.Close()

	conn, err := Connect(box.Addr(), &Options{
		DefaultSpace:        "tester",
		User:                "tester",
		Password:            "12345678",
		ResultUnmarshalMode: ResultAsRawData,
	})
	require.NoError(err)
	defer conn.Close()

	res := conn.Exec(context.Background(), &Select{
		Key:   33,
		Index: "tester_id",
	})
	require.NoError(res.Error)
	assert.Nil(res.Data)
	assert.Equal([]interface{}{[]interface{}{int64(33), int64(45)}}, res.RawData)

	tuples, err := conn.Execute(&Select{
		Key:   33,
		Index: "tester_id",
	})
	require.NoError(err)
	assert.Equal([][]interface{}{{int64(33), int64(45)}}, tuples)
}
