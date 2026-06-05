package meta

func (d *DB) SetPluginConfig(pluginName, key, value string) error {
	_, err := d.db.Exec(`INSERT INTO plugin_configs (plugin_name, key, value) VALUES (?, ?, ?) 
		ON CONFLICT(plugin_name, key) DO UPDATE SET value=excluded.value`, pluginName, key, value)
	return err
}

func (d *DB) GetPluginConfig(pluginName, key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM plugin_configs WHERE plugin_name = ? AND key = ?`, pluginName, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}
