package gorp

//++ TODO v2-phase3: HasPostGet > PostGetter, HasPostDelete > PostDeleter, etc.

// PostUpdate() will be executed after the GET statement.
type HasPostGet interface {
	PostGet(SqlExecutor) error
}

// PostUpdate() will be executed after the DELETE statement
type HasPostDelete interface {
	PostDelete(SqlExecutor) error
}

// PostUpdate() will be executed after the UPDATE statement
type HasPostUpdate interface {
	PostUpdate(SqlExecutor) error
}

// PostInsert() will be executed after the INSERT statement
type HasPostInsert interface {
	PostInsert(SqlExecutor) error
}

// PreDelete() will be executed before the DELETE statement.
type HasPreDelete interface {
	PreDelete(SqlExecutor) error
}

// PreUpdate() will be executed before UPDATE statement.
type HasPreUpdate interface {
	PreUpdate(SqlExecutor) error
}

// PreInsert() will be executed before INSERT statement.
type HasPreInsert interface {
	PreInsert(SqlExecutor) error
}
