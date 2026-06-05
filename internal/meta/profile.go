package meta

import (
	"database/sql"
	"fmt"
)

type Profile struct {
	ID         int
	Name       string
	ConfigText string
	Customer   sql.NullString
	Server     sql.NullString
	Project    sql.NullString
}

func (d *DB) GetTmuxProfile(name string) (string, error) {
	var config string
	err := d.db.QueryRow(`SELECT config_text FROM tmux_profiles WHERE name = ?`, name).Scan(&config)
	return config, err
}

func (d *DB) GetProfileMetadata(name string) (*Profile, error) {
	var p Profile
	err := d.db.QueryRow(`SELECT id, name, config_text, customer, server, project FROM tmux_profiles WHERE name = ?`, name).
		Scan(&p.ID, &p.Name, &p.ConfigText, &p.Customer, &p.Server, &p.Project)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query profile: %w", err)
	}
	return &p, nil
}

func (d *DB) UpdateProfileMetadata(name string, customer, server, project sql.NullString) error {
	_, err := d.db.Exec(`UPDATE tmux_profiles SET customer = ?, server = ?, project = ? WHERE name = ?`, customer, server, project, name)
	if err != nil {
		return fmt.Errorf("failed to update profile metadata: %w", err)
	}
	return nil
}

func (d *DB) UpdateProfileConfig(name, configText string) error {
	_, err := d.db.Exec(`UPDATE tmux_profiles SET config_text = ? WHERE name = ?`, configText, name)
	if err != nil {
		return fmt.Errorf("failed to update profile config: %w", err)
	}
	return nil
}

func (d *DB) CreateProfile(name string) error {
	_, err := d.db.Exec(`INSERT INTO tmux_profiles (name, config_text) SELECT ?, config_text FROM tmux_profiles WHERE name = 'default'`, name)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}
	return nil
}

func (d *DB) ListProfiles() ([]Profile, error) {
	rows, err := d.db.Query(`SELECT id, name, config_text, customer, server, project FROM tmux_profiles ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.ConfigText, &p.Customer, &p.Server, &p.Project); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}
