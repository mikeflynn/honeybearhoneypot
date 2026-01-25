package entity

const MigrationFixExecLoginSource = `
	UPDATE events
	SET source = 'user'
	WHERE type = 'login' AND source == 'system';
`
