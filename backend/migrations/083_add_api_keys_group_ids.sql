-- 为 API Key 增加多分组绑定能力。
-- group_id 保留为兼容/主分组字段；
-- group_ids 保存完整的有序分组列表，用于负载均衡与故障转移。

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS group_ids BIGINT[] NOT NULL DEFAULT '{}';

-- 将已有单 group_id 数据回填到 group_ids。
UPDATE api_keys
SET group_ids = ARRAY[group_id]
WHERE group_id IS NOT NULL
  AND cardinality(COALESCE(group_ids, ARRAY[]::bigint[])) = 0;

COMMENT ON COLUMN api_keys.group_ids IS 'Ordered group ids bound to this API key for load balancing / failover';

CREATE INDEX IF NOT EXISTS idx_api_keys_group_ids_gin
ON api_keys USING GIN (group_ids);
