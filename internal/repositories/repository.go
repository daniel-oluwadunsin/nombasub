package repositories

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository[T any] struct {
	db *gorm.DB
}

type Select []string

type OrderBy struct {
	Column string
	Desc   bool
	Asc    bool
}

type Preload struct {
	Association string
	Select      []string
	Limit       *int
	OrderBy     []OrderBy
	Condition   string
	Args        []interface{}
}

type FindArgs struct {
	Filter   *QueryFilter
	Joins    []Join
	Preloads []Preload
	Select   Select
	Limit    *int
	OrderBy  []OrderBy
}

func New[T any](db *gorm.DB, tableName string) *Repository[T] {
	db = db.Table(tableName)
	return &Repository[T]{db}
}

func loadDbWithArgs(db *gorm.DB, args *FindArgs, withoutPreloading bool) *gorm.DB {
	if args != nil {
		for _, join := range args.Joins {
			args := []interface{}{}

			args = append(args, join.args...)

			db = db.Joins(join.column, args...)
		}

		if !withoutPreloading {
			for _, preload := range args.Preloads {
				db = db.Preload(preload.Association, func(db *gorm.DB) *gorm.DB {
					if preload.Condition != "" {
						db = db.Where(preload.Condition, preload.Args...)
					}

					if len(preload.Select) != 0 {
						db = db.Select(preload.Select)
					}

					if len(preload.OrderBy) != 0 {
						for _, order := range preload.OrderBy {
							if order.Desc {
								db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: order.Column}, Desc: true})
							}
							if order.Asc {
								db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: order.Column}, Desc: false})
							}
						}
					}

					if preload.Limit != nil && *preload.Limit > 0 {
						db = db.Limit(*preload.Limit)
					}

					return db
				})
			}
		}

		if args.Filter != nil && args.Filter.query != "" {
			db = db.Where(args.Filter.query, args.Filter.args...)
		}

		if len(args.Select) != 0 {
			db = db.Select([]string(args.Select))
		}

		if len(args.OrderBy) != 0 {
			for _, order := range args.OrderBy {
				if order.Desc {
					db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: order.Column}, Desc: true})
				}
				if order.Asc {
					db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: order.Column}, Desc: false})
				}
			}
		}

		if args.Limit != nil && *args.Limit > 0 {
			db = db.Limit(*args.Limit)
		}
	}

	return db
}

func (r *Repository[T]) FindRaw(args *FindArgs) (*T, error) {
	db := r.db.Session(&gorm.Session{})

	var value T

	db = loadDbWithArgs(db, args, false)

	if result := db.Limit(1).Find(&value); result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected == 0 {
		return nil, nil
	}

	return &value, nil
}

func (r *Repository[T]) ExistsRaw(args *FindArgs) (bool, error) {
	val, err := r.FindRaw(args)

	if err != nil {
		return false, err
	}

	if val == nil {
		return false, nil
	}

	return true, nil
}

func (r *Repository[T]) Find(model *T, args *FindArgs) (*T, error) {
	db := r.db.Session(&gorm.Session{})

	var value T

	db = loadDbWithArgs(db, args, false)

	if result := db.Where(model).Limit(1).Find(&value); result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected == 0 {
		return nil, nil
	}

	return &value, nil
}

func (r *Repository[T]) FindById(id interface{}, args *FindArgs) (*T, error) {
	db := r.db.Session(&gorm.Session{})

	var value T

	db = loadDbWithArgs(db, args, false)

	if result := db.Where("id = ?", id).Limit(1).Find(&value); result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected == 0 {
		return nil, nil
	}

	return &value, nil
}

func (r *Repository[T]) Exists(model *T, args *FindArgs) (bool, error) {
	val, err := r.Find(model, args)

	if err != nil {
		return false, err
	}

	if val == nil {
		return false, nil
	}

	return true, nil
}

func (r *Repository[T]) FindManyRaw(args *FindArgs) ([]T, error) {
	db := r.db.Session(&gorm.Session{})

	var values []T

	db = loadDbWithArgs(db, args, false)

	if result := db.Find(&values); result.Error != nil {
		return nil, result.Error
	}

	return values, nil
}

func (r *Repository[T]) FindMany(model *T, args *FindArgs) ([]T, error) {
	db := r.db.Session(&gorm.Session{})

	var values []T

	db = loadDbWithArgs(db, args, false)

	if result := db.Where(model).Find(&values); result.Error != nil {
		return nil, result.Error
	}

	return values, nil
}

func (r *Repository[T]) FindManyPaginated(model *T, args *FindArgs, pg *requests.PaginationQuery) (*responses.PaginatedResponse[T], error) {
	db := r.db.Session(&gorm.Session{})

	var values []T

	page := 0
	limit := 0
	var count int64 = 0

	if pg.Limit != nil {
		limit = *pg.Limit
	} else {
		limit = 10
	}

	if pg.Page != nil {
		page = *pg.Page
	} else {
		page = 1
	}
	skip := (page - 1) * limit

	dbClone := loadDbWithArgs(db, args, true)
	if countResult := dbClone.Where(model).Count(&count); countResult.Error != nil {
		return nil, countResult.Error
	}

	db = loadDbWithArgs(db, args, false)

	if result := db.Where(model).Offset(skip).Limit(limit).Find(&values); result.Error != nil {
		return nil, result.Error
	}

	return responses.NewPaginatedResponse(page, limit, int(count), values), nil
}

func (r *Repository[T]) FindManyPaginatedRaw(args *FindArgs, pg *requests.PaginationQuery) (*responses.PaginatedResponse[T], error) {
	db := r.db.Session(&gorm.Session{})

	var values []T

	args.Limit = nil

	page := 0
	limit := 0
	var count int64 = 0

	if pg.Limit != nil {
		limit = *pg.Limit
	} else {
		limit = 10
	}

	if pg.Page != nil {
		page = *pg.Page
	} else {
		page = 1
	}
	skip := (page - 1) * limit

	dbClone := loadDbWithArgs(db, args, true)
	if countResult := dbClone.Count(&count); countResult.Error != nil {
		return nil, countResult.Error
	}

	db = loadDbWithArgs(db, args, false)
	if result := db.Offset(skip).Limit(limit).Find(&values); result.Error != nil {
		return nil, result.Error
	}

	return responses.NewPaginatedResponse(page, limit, int(count), values), nil
}

func (r *Repository[T]) Count(args *FindArgs) (int64, error) {
	db := r.db.Session(&gorm.Session{})

	var count int64

	db = loadDbWithArgs(db, args, false)

	if result := db.Count(&count); result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

func (r *Repository[T]) Create(data *T) (*T, error) {
	db := r.db.Session(&gorm.Session{})

	if data == nil {
		return nil, gorm.ErrInvalidData
	}

	result := db.Create(data)

	if result.Error != nil {
		return nil, result.Error
	}

	return data, nil
}

func (r *Repository[T]) CreateMany(data []T) ([]T, error) {
	db := r.db.Session(&gorm.Session{})

	if len(data) == 0 {
		return nil, gorm.ErrInvalidData
	}

	result := db.Create(&data)

	if result.Error != nil {
		return nil, result.Error
	}

	return data, nil
}

func (r *Repository[T]) Update(data *T) (*T, error) {
	db := r.db.Session(&gorm.Session{NewDB: true})

	result := db.Select("*").Updates(data)

	if result.Error != nil {
		return nil, result.Error
	}

	return data, nil
}

func (r *Repository[T]) Delete(args *FindArgs) error {
	db := r.db.Session(&gorm.Session{NewDB: true})

	db = loadDbWithArgs(db, args, false)

	result := db.Delete(new(T))

	if result.Error != nil {
		return result.Error
	}

	return nil
}
