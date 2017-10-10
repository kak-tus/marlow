package models

import "os"
import "fmt"
import "strings"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow"

func addAuthorRow(db *sql.DB, values ...[]string) error {
	for _, rowValues := range values {
		valueString := strings.Join(rowValues, ",")
		statement := fmt.Sprintf("insert into authors (id,name) values(%s);", valueString)
		r, e := db.Exec(statement)

		if e != nil {
			return e
		}

		count, e := r.RowsAffected()

		if e != nil {
			return e
		}

		if count != 1 {
			return fmt.Errorf("no-rows-created")
		}
	}

	return nil
}

func Test_Author(t *testing.T) {
	g := goblin.Goblin(t)
	var db *sql.DB
	var store *AuthorStore

	dbFile := "author-testing.db"

	g.Describe("AuthorBlueprint test suite", func() {

		g.It("results in empty string w/o values", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{})
			g.Assert(r).Equal("")
		})

		g.It("supports null values on sql.NullInt64 fields", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Valid: false},
				},
			})
			g.Assert(r).Equal("WHERE authors.university_id IS NULL")
		})

		g.It("supports range on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				IDRange: []int{1, 2},
			})

			g.Assert(r).Equal("WHERE authors.id > 1 AND authors.id < 2")
		})

		g.It("supports 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{ID: []int{1, 2, 3}})
			g.Assert(r).Equal("WHERE authors.id IN ('1','2','3')")
		})

		g.It("supports a combination of range and 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				ID:      []int{1, 2, 3},
				IDRange: []int{1, 4},
			})

			g.Assert(r).Equal("WHERE authors.id IN ('1','2','3') AND authors.id > 1 AND authors.id < 4")
		})

	})

	g.Describe("Author model & generated store test suite", func() {

		g.Before(func() {
			var e error
			db, e = loadDB(dbFile)
			g.Assert(e).Equal(nil)

			authors := [][]string{}

			for i := 1; i < 150; i++ {
				id, name := fmt.Sprintf("%d", i), fmt.Sprintf("'author-%d'", (i*10)+1)
				authors = append(authors, []string{id, name})
			}

			g.Assert(addAuthorRow(db, authors...)).Equal(nil)

			_, e = db.Exec("insert into authors (id,name,university_id) values(1337,'learned author',10);")
			g.Assert(e).Equal(nil)
			_, e = db.Exec("insert into authors (id,name,university_id) values(1338,'other author',null);")
			g.Assert(e).Equal(nil)
		})

		g.BeforeEach(func() {
			store = &AuthorStore{DB: db}
		})

		g.After(func() {
			e := db.Close()
			g.Assert(e).Equal(nil)
			os.Remove(dbFile)
		})

		g.It("allows the consumer to search for authors w/o (default limit)", func() {
			authors, e := store.FindAuthors(nil)
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(marlow.DefaultBlueprintLimit)
		})

		g.It("allows the consumer to search for authors w/ blueprint (explicit limit)", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{Limit: 20})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(20)
		})

		g.It("allows the consumer to search for authors by explicit Name", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				Name: []string{"author-11", "author-21", "not-exists"},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(2)
		})

		g.It("allows the consumer to search by 'NameLike'", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				NameLike: []string{"%-100%"},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(1)
		})

		g.It("allows the consumer to search for authors by explicit ID", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(2)
		})

		g.It("correctly serializes null/not null values into a sql.NullInt64 field", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				ID: []int{1337, 1338},
			})
			g.Assert(e).Equal(nil)

			g.Assert(authors[0].Name).Equal("learned author")
			g.Assert(authors[0].UniversityID.Valid).Equal(true)
			g.Assert(authors[0].UniversityID.Int64).Equal(10)

			g.Assert(authors[1].Name).Equal("other author")
			g.Assert(authors[1].UniversityID.Valid).Equal(false)
		})

		g.It("allows consumer to search by authors with null UniversityID", func() {
			_, e := store.CountAuthors(&AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Valid: false},
				},
			})
			g.Assert(e).Equal(nil)
		})

		g.It("allows the consumer to count authors by blueprint", func() {
			count, e := store.CountAuthors(&AuthorBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(2)
		})

	})
}
