// Package store provides typed CRUD operations for [model.Model] types.
//
// Build a store with [New] and call its methods inside a [conn.Tx].
//
// Example:
//
//	crud := store.New[*User]("users")
//
//	tx, err := db.BeginTx(ctx, nil)
//	if err != nil {
//		return err
//	}
//	defer tx.Rollback()
//
//	user := &User{Name: "Ada"}
//	_, err = crud.Create(tx, user)
//
//	loaded, err := crud.Retrieve(tx, sql.Named("id", user.ID))
//	err = crud.Update(tx, user)
//	_, err = crud.Delete(tx, sql.Named("id", user.ID))
//
//	cursor, err := crud.List(tx, (&filter.Filter{}).OrderBy("name").Limit(10))
//	users, err := cursor.List()
package store
