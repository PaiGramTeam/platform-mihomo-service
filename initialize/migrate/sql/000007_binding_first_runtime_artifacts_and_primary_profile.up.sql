SET @multiple_default_profile_bindings = (
    SELECT COUNT(*)
    FROM (
        SELECT binding_id
        FROM account_profiles
        WHERE is_default = TRUE
        GROUP BY binding_id
        HAVING COUNT(*) > 1
    ) duplicates
);
SET @default_profile_precheck = IF(
    @multiple_default_profile_bindings > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000007 failed: multiple default account_profiles rows for binding_id''',
    'DO 0'
);
PREPARE default_profile_precheck_stmt FROM @default_profile_precheck;
EXECUTE default_profile_precheck_stmt;
DEALLOCATE PREPARE default_profile_precheck_stmt;

SET @unmapped_runtime_artifacts = (
    SELECT COUNT(*)
    FROM runtime_artifacts ra
    LEFT JOIN credential_records cr
        ON cr.platform_account_id = ra.platform_account_id
    WHERE cr.binding_id IS NULL
);
SET @runtime_artifact_precheck = IF(
    @unmapped_runtime_artifacts > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000007 failed: runtime_artifacts rows without credential binding_id mapping''',
    'DO 0'
);
PREPARE runtime_artifact_precheck_stmt FROM @runtime_artifact_precheck;
EXECUTE runtime_artifact_precheck_stmt;
DEALLOCATE PREPARE runtime_artifact_precheck_stmt;

SET @duplicate_binding_runtime_artifacts = (
    SELECT COUNT(*)
    FROM (
        SELECT cr.binding_id, ra.artifact_type, ra.scope_key
        FROM runtime_artifacts ra
        JOIN credential_records cr
            ON cr.platform_account_id = ra.platform_account_id
        GROUP BY cr.binding_id, ra.artifact_type, ra.scope_key
        HAVING COUNT(*) > 1
    ) duplicates
);
SET @runtime_artifact_duplicate_precheck = IF(
    @duplicate_binding_runtime_artifacts > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000007 failed: duplicate runtime_artifacts rows for binding_id, artifact_type, scope_key''',
    'DO 0'
);
PREPARE runtime_artifact_duplicate_precheck_stmt FROM @runtime_artifact_duplicate_precheck;
EXECUTE runtime_artifact_duplicate_precheck_stmt;
DEALLOCATE PREPARE runtime_artifact_duplicate_precheck_stmt;

ALTER TABLE account_profiles
    ADD COLUMN default_profile_marker TINYINT GENERATED ALWAYS AS (IF(is_default = TRUE, 1, NULL)) STORED AFTER is_default,
    ADD UNIQUE KEY uniq_default_profile_per_binding (binding_id, default_profile_marker);

ALTER TABLE runtime_artifacts
    ADD COLUMN binding_id BIGINT UNSIGNED NULL AFTER id,
    ADD KEY idx_runtime_binding_id (binding_id);

UPDATE runtime_artifacts ra
JOIN credential_records cr
    ON cr.platform_account_id = ra.platform_account_id
SET ra.binding_id = cr.binding_id;

ALTER TABLE runtime_artifacts
    DROP INDEX uniq_runtime_artifact,
    MODIFY binding_id BIGINT UNSIGNED NOT NULL,
    ADD UNIQUE KEY uniq_runtime_artifact_binding (binding_id, artifact_type, scope_key);
