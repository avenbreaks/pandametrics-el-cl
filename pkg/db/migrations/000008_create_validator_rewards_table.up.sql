CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_epoch INT,
		f_balance_eth REAL,
		f_reward BIGINT,
		f_max_reward INT,
		f_max_att_reward INT,
		f_max_sync_reward INT,
		f_att_slot INT,
		f_base_reward INT,
		f_in_sync_committee BOOL,
		f_missing_source BOOL,
		f_missing_target BOOL, 
		f_missing_head BOOL,
		f_status SMALLINT,
		CONSTRAINT t_validator_rewards_summary_pkey PRIMARY KEY (f_val_idx,f_epoch));