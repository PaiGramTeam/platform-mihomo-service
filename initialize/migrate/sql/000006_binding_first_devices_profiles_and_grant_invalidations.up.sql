ALTER TABLE device_records
    ADD COLUMN binding_id BIGINT UNSIGNED NULL AFTER id;

UPDATE device_records d
JOIN credential_records c ON c.platform_account_id = d.platform_account_id
SET d.binding_id = c.binding_id
WHERE d.binding_id IS NULL;

DELETE d FROM device_records d
LEFT JOIN credential_records c ON c.platform_account_id = d.platform_account_id
WHERE c.id IS NULL;

ALTER TABLE device_records
    MODIFY COLUMN binding_id BIGINT UNSIGNED NOT NULL,
    DROP INDEX uniq_device_record,
    ADD UNIQUE KEY uniq_device_record_binding (binding_id, device_id),
    ADD KEY idx_device_binding_id (binding_id);

DELETE p FROM account_profiles p
JOIN account_profiles keep
  ON keep.binding_id = p.binding_id
 AND keep.player_id = p.player_id
 AND keep.region = p.region
 AND keep.id < p.id;

ALTER TABLE account_profiles
    DROP INDEX uniq_platform_profile,
    ADD UNIQUE KEY uniq_profile_binding_player_region (binding_id, player_id, region);

CREATE TABLE IF NOT EXISTS consumer_grant_invalidations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    binding_id BIGINT UNSIGNED NOT NULL,
    consumer VARCHAR(64) NOT NULL,
    minimum_grant_version BIGINT UNSIGNED NOT NULL,
    invalidated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_consumer_grant_invalidations_binding_consumer (binding_id, consumer)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
