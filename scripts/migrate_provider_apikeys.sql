-- ============================================================
-- Migration: Convert provider.api_keys from string array to object array
-- FROM: ["sk-aaaa", "sk-bbbb"]
-- TO:   [{"value": "sk-aaaa", "description": ""}, {"value": "sk-bbbb", "description": ""}]
-- ============================================================

-- Step 1: Preview — check current data before migration
SELECT id, code, name, api_keys
FROM provider
WHERE api_keys IS NOT NULL
  AND JSON_LENGTH(api_keys) > 0;

-- Step 2: Execute migration
UPDATE provider
SET api_keys = (
    SELECT JSON_ARRAYAGG(
        JSON_OBJECT('value', jt.val, 'description', '')
    )
    FROM JSON_TABLE(api_keys, '$[*]' COLUMNS (val VARCHAR(512) PATH '$')) AS jt
)
WHERE api_keys IS NOT NULL
  AND JSON_LENGTH(api_keys) > 0
  AND deleted = '0';

-- Step 3: Verify — check data after migration
SELECT id, code, name, api_keys
FROM provider
WHERE api_keys IS NOT NULL
  AND JSON_LENGTH(api_keys) > 0;
