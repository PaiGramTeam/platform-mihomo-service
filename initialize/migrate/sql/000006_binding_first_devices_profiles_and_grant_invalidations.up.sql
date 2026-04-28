SET @duplicate_profile_rows = (
    SELECT COUNT(*)
    FROM (
        SELECT binding_id, player_id, region
        FROM account_profiles
        GROUP BY binding_id, player_id, region
        HAVING COUNT(*) > 1
    ) duplicates
);
SET @profile_duplicate_precheck = IF(
    @duplicate_profile_rows > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000006 failed: duplicate account_profiles rows for binding_id, player_id, region''',
    'DO 0'
);
PREPARE profile_duplicate_precheck_stmt FROM @profile_duplicate_precheck;
EXECUTE profile_duplicate_precheck_stmt;
DEALLOCATE PREPARE profile_duplicate_precheck_stmt;

ALTER TABLE device_records
    ADD COLUMN binding_id BIGINT UNSIGNED NULL AFTER id;

UPDATE device_records d
JOIN credential_records c ON c.platform_account_id = d.platform_account_id
SET d.binding_id = c.binding_id
WHERE d.binding_id IS NULL;

SET @device_backfill_failures = (
    SELECT COUNT(*)
    FROM device_records
    WHERE binding_id IS NULL
);
SET @device_backfill_precheck = IF(
    @device_backfill_failures > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000006 failed: device_records rows cannot be backfilled from credential_records''',
    'DO 0'
);
PREPARE device_backfill_precheck_stmt FROM @device_backfill_precheck;
EXECUTE device_backfill_precheck_stmt;
DEALLOCATE PREPARE device_backfill_precheck_stmt;

ALTER TABLE device_records
    MODIFY COLUMN binding_id BIGINT UNSIGNED NOT NULL,
    DROP INDEX uniq_device_record,
    ADD UNIQUE KEY uniq_device_record_binding (binding_id, device_id),
    ADD KEY idx_device_binding_id (binding_id);

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
