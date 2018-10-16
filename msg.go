package vbdb

// InsertMessage inserts the passed string into the msg table and
// returns the auto-incremented id (foreign-key in other tables).
func InsertMessage(msg string) (id int64, err error) {
	return s.ExecID("INSERT INTO msg(message) VALUES(?)", msg)
}

// SelectMessage receives the message associated to the passed
// id.
func SelectMessage(id int64) (msg string, exists bool, err error) {
	exists, err = s.SelectExists("SELECT message FROM msg WHERE id=?", []interface{}{id}, []interface{}{&msg})
	return
}

// ExistsMessage checks wheter the id (and therefore) it's
// associated value exist or not.
func ExistsMessage(id int64) (exists bool, err error) {
	return s.MysqlExists("SELECT id FROM msg WHERE id=?", id)
}

// SelectMessageExists checks whether the id (and therefore) it's
// associated value exists or not and if so also returns the value.
// If the id doesn't exists the msg pointer will be nil.
func SelectMessageExists(id int64) (msg *string, err error) {
	exists, err := s.SelectExists("SELECT message FROM msg WHERE id=?", []interface{}{id}, []interface{}{msg})
	if err != nil || !exists {
		return nil, err
	}
	return msg, err
}
