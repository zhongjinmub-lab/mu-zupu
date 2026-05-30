-- 智能体族谱SAAS - 回滚 Agent 与知识库绑定

ALTER TABLE agents
    DROP CONSTRAINT IF EXISTS chk_agents_status;

DROP INDEX IF EXISTS idx_agent_kb_bindings_kb;
DROP INDEX IF EXISTS idx_agent_kb_bindings_agent;
DROP TABLE IF EXISTS agent_knowledge_bases;
