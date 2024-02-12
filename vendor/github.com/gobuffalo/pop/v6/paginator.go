package pop

import (
	"encoding/json"
	"strconv"

	"github.com/gobuffalo/pop/v6/internal/defaults"
)

// PaginatorPerPageDefault is the amount of results per page
var PaginatorPerPageDefault = 20

// PaginatorPageKey is the query parameter holding the current page index
var PaginatorPageKey = "page"

// PaginatorPerPageKey is the query parameter holding the amount of results per page
// to override the default one
var PaginatorPerPageKey = "per_page"

type paginable interface {
	Paginate() string
}

var _ paginable = Paginator{}

// Paginator is a type used to represent the pagination of records
// from the database.
type Paginator struct {
	// Current page you're on
	Page int `json:"page"`
	// Number of results you want per page
	PerPage int `json:"per_page"`
	// Page * PerPage (ex: 2 * 20, Offset == 40)
	Offset int `json:"offset"`
	// Total potential records matching the query
	TotalEntriesSize int `json:"total_entries_size"`
	// Total records returns, will be <= PerPage
	CurrentEntriesSize int `json:"current_entries_size"`
	// Total pages
	TotalPages int `json:"total_pages"`
}

// Paginate implements the paginable interface.
func (p Paginator) Paginate() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func (p Paginator) String() string {
	return p.Paginate()
}

// NewPaginator returns a new `Paginator` value with the appropriate
// defaults set.
func NewPaginator(page int, perPage int) *Paginator {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	p := &Paginator{Page: page, PerPage: perPage}
	p.Offset = (page - 1) * p.PerPage
	return p
}

// PaginationParams is a parameters provider interface to get the pagination params from
type PaginationParams interface {
	Get(key string) string
}

// NewPaginatorFromParams takes an interface of type `PaginationParams`,
// the `url.Values` type works great with this interface, and returns
// a new `Paginator` based on the params or `PaginatorPageKey` and
// `PaginatorPerPageKey`. Defaults are `1` for the page and
// PaginatorPerPageDefault for the per page value.
func NewPaginatorFromParams(params PaginationParams) *Paginator {
	page := defaults.String(params.Get(PaginatorPageKey), "1")

	perPage := defaults.String(params.Get(PaginatorPerPageKey), strconv.Itoa(PaginatorPerPageDefault))

	p, err := strconv.Atoi(page)
	if err != nil {
		p = 1
	}

	pp, err := strconv.Atoi(perPage)
	if err != nil {
		pp = PaginatorPerPageDefault
	}
	return NewPaginator(p, pp)
}

// Paginate records returned from the database.
//
//	q := c.Paginate(2, 15)
//	q.All(&[]User{})
//	q.Paginator
func (c *Connection) Paginate(page int, perPage int) *Query {
	return Q(c).Paginate(page, perPage)
}

// Paginate records returned from the database.
//
//	q = q.Paginate(2, 15)
//	q.All(&[]User{})
//	q.Paginator
func (q *Query) Paginate(page int, perPage int) *Query {
	q.Paginator = NewPaginator(page, perPage)
	return q
}

// PaginateFromParams paginates records returned from the database.
//
//	q := c.PaginateFromParams(req.URL.Query())
//	q.All(&[]User{})
//	q.Paginator
func (c *Connection) PaginateFromParams(params PaginationParams) *Query {
	return Q(c).PaginateFromParams(params)
}

// PaginateFromParams paginates records returned from the database.
//
//	q = q.PaginateFromParams(req.URL.Query())
//	q.All(&[]User{})
//	q.Paginator
func (q *Query) PaginateFromParams(params PaginationParams) *Query {
	q.Paginator = NewPaginatorFromParams(params)
	return q
}
