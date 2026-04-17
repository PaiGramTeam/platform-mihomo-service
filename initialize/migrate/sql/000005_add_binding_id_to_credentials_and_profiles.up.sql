ALTER TABLE credential_records
    ADD COLUMN binding_id BIGINT UNSIGNED NULL AFTER id;

UPDATE credential_records
SET binding_id = CASE
    WHEN platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 2), '_', -1) AS UNSIGNED)
    WHEN platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 3), '_', -1) AS UNSIGNED)
END
WHERE binding_id IS NULL;

ALTER TABLE credential_records
    MODIFY COLUMN binding_id BIGINT UNSIGNED NOT NULL,
    ADD UNIQUE KEY uniq_credential_binding_id (binding_id);

ALTER TABLE account_profiles
    ADD COLUMN binding_id BIGINT UNSIGNED NULL AFTER id;

UPDATE account_profiles
SET binding_id = CASE
    WHEN platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 2), '_', -1) AS UNSIGNED)
    WHEN platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 3), '_', -1) AS UNSIGNED)
END
WHERE binding_id IS NULL;

ALTER TABLE account_profiles
    MODIFY COLUMN binding_id BIGINT UNSIGNED NOT NULL,
    ADD KEY idx_profile_binding_id (binding_id);
