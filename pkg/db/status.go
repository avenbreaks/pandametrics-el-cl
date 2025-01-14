package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

// Postgres intregration variables
var (
	UpsertStatus = `
	INSERT INTO t_status (
		f_id, 
		f_status)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT t_status_pkey
		DO NOTHING
	`
)
