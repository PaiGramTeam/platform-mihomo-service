SET @invalid_credential_account_ids = (
    SELECT COUNT(*)
    FROM credential_records
    WHERE platform_account_id IS NULL
       OR NOT (
           platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$'
           OR platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$'
       )
);
SET @credential_account_id_precheck = IF(
    @invalid_credential_account_ids > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000005 failed: malformed credential_records platform_account_id values''',
    'DO 0'
);
PREPARE credential_account_id_precheck_stmt FROM @credential_account_id_precheck;
EXECUTE credential_account_id_precheck_stmt;
DEALLOCATE PREPARE credential_account_id_precheck_stmt;

SET @duplicate_parsed_credential_binding_ids = (
    SELECT COUNT(*)
    FROM (
        SELECT CASE
            WHEN platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 2), '_', -1) AS UNSIGNED)
            WHEN platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$' THEN CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(platform_account_id, '_', 3), '_', -1) AS UNSIGNED)
        END AS parsed_binding_id
        FROM credential_records
        WHERE platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$'
           OR platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$'
        GROUP BY parsed_binding_id
        HAVING COUNT(*) > 1
    ) duplicate_credential_binding_ids
);
SET @duplicate_credential_binding_id_precheck = IF(
    @duplicate_parsed_credential_binding_ids > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000005 failed: duplicate parsed credential_records binding_id values''',
    'DO 0'
);
PREPARE duplicate_credential_binding_id_precheck_stmt FROM @duplicate_credential_binding_id_precheck;
EXECUTE duplicate_credential_binding_id_precheck_stmt;
DEALLOCATE PREPARE duplicate_credential_binding_id_precheck_stmt;

SET @invalid_profile_account_ids = (
    SELECT COUNT(*)
    FROM account_profiles
    WHERE platform_account_id IS NULL
       OR NOT (
           platform_account_id REGEXP '^binding_[1-9][0-9]*_.+$'
           OR platform_account_id REGEXP '^hoyo_ref_[1-9][0-9]*_.+$'
       )
);
SET @profile_account_id_precheck = IF(
    @invalid_profile_account_ids > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000005 failed: malformed account_profiles platform_account_id values''',
    'DO 0'
);
PREPARE profile_account_id_precheck_stmt FROM @profile_account_id_precheck;
EXECUTE profile_account_id_precheck_stmt;
DEALLOCATE PREPARE profile_account_id_precheck_stmt;

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
