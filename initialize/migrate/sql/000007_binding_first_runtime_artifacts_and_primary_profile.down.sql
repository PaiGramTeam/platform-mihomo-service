SET @duplicate_platform_runtime_artifacts = (
    SELECT COUNT(*)
    FROM (
        SELECT platform_account_id, artifact_type, scope_key
        FROM runtime_artifacts
        GROUP BY platform_account_id, artifact_type, scope_key
        HAVING COUNT(*) > 1
    ) duplicates
);
SET @runtime_artifact_rollback_precheck = IF(
    @duplicate_platform_runtime_artifacts > 0,
    'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''migration 000007 rollback failed: runtime_artifacts rows would violate platform_account_id uniqueness''',
    'DO 0'
);
PREPARE runtime_artifact_rollback_precheck_stmt FROM @runtime_artifact_rollback_precheck;
EXECUTE runtime_artifact_rollback_precheck_stmt;
DEALLOCATE PREPARE runtime_artifact_rollback_precheck_stmt;

ALTER TABLE runtime_artifacts
    DROP INDEX uniq_runtime_artifact_binding,
    DROP INDEX idx_runtime_binding_id,
    DROP COLUMN binding_id,
    ADD UNIQUE KEY uniq_runtime_artifact (platform_account_id, artifact_type, scope_key);

ALTER TABLE account_profiles
    DROP INDEX uniq_default_profile_per_binding,
    DROP COLUMN default_profile_marker;
