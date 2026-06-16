package tenant

// Permission 表示一项系统权限。
type Permission struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Module      string `json:"module"`
	Description string `json:"description"`
}

// RolePermissionMatrix 表示角色与权限的能力矩阵。
type RolePermissionMatrix struct {
	Roles       []string            `json:"roles"`
	Permissions []Permission        `json:"permissions"`
	Matrix      map[string][]string `json:"matrix"`
}

// DefaultPermissions 返回系统内置权限目录。
func DefaultPermissions() []Permission {
	return []Permission{
		{Code: "agent.create", Name: "创建智能体", Module: "agent", Description: "创建 Agent"},
		{Code: "agent.edit", Name: "编辑智能体", Module: "agent", Description: "编辑 Agent 配置与提示词"},
		{Code: "agent.publish", Name: "发布智能体", Module: "agent", Description: "发布/回滚 Agent"},
		{Code: "agent.delete", Name: "归档智能体", Module: "agent", Description: "归档 Agent"},
		{Code: "agent.chat", Name: "对话", Module: "agent", Description: "使用 Agent 对话"},
		{Code: "kb.manage", Name: "管理知识库", Module: "kb", Description: "创建/编辑/删除知识库与文档"},
		{Code: "kb.search", Name: "检索知识库", Module: "kb", Description: "执行向量/混合检索"},
		{Code: "workflow.manage", Name: "管理工作流", Module: "workflow", Description: "创建/编辑/发布/归档/复制/运行工作流"},
		{Code: "workflow.view", Name: "查看工作流", Module: "workflow", Description: "查看工作流列表与详情"},
		{Code: "channel.manage", Name: "管理渠道", Module: "channel", Description: "创建/启用/禁用/归档/复制渠道"},
		{Code: "channel.view", Name: "查看渠道", Module: "channel", Description: "查看渠道列表与接入代码"},
		{Code: "plugin.manage", Name: "管理插件", Module: "plugin", Description: "启用/禁用插件"},
		{Code: "billing.view", Name: "查看计费", Module: "billing", Description: "查看用量、订单与套餐"},
		{Code: "billing.manage", Name: "管理计费", Module: "billing", Description: "订阅变更、支付操作"},
		{Code: "license.manage", Name: "管理授权", Module: "license", Description: "创建/吊销/验证 License"},
		{Code: "tenant.members", Name: "管理成员", Module: "tenant", Description: "邀请/移除成员、修改角色"},
		{Code: "tenant.settings", Name: "租户设置", Module: "tenant", Description: "修改租户信息与配置"},
		{Code: "audit.view", Name: "查看审计", Module: "audit", Description: "查看/导出审计日志"},
		{Code: "webhook.manage", Name: "管理 Webhook", Module: "webhook", Description: "配置/测试/重试 Webhook"},
	}
}

// DefaultRoles 返回系统内置角色列表。
func DefaultRoles() []string {
	return []string{"owner", "admin", "member", "viewer"}
}

// DefaultRolePermissionMatrix 返回内置角色与权限的能力矩阵，纯函数。
func DefaultRolePermissionMatrix() RolePermissionMatrix {
	allPerms := DefaultPermissions()
	codes := make([]string, 0, len(allPerms))
	for _, p := range allPerms {
		codes = append(codes, p.Code)
	}
	// owner/admin 拥有全部权限
	ownerPerms := codes
	adminPerms := codes
	// member（writer）拥有除 tenant.members/tenant.settings/billing.manage/license.manage 以外的权限
	memberExclude := map[string]bool{
		"tenant.members":  true,
		"tenant.settings": true,
		"billing.manage":  true,
		"license.manage":  true,
	}
	memberPerms := make([]string, 0)
	for _, c := range codes {
		if !memberExclude[c] {
			memberPerms = append(memberPerms, c)
		}
	}
	// viewer 仅拥有 view/search/chat 类权限
	viewerPerms := []string{
		"agent.chat",
		"kb.search",
		"workflow.view",
		"channel.view",
		"billing.view",
		"audit.view",
	}
	return RolePermissionMatrix{
		Roles:       DefaultRoles(),
		Permissions: allPerms,
		Matrix: map[string][]string{
			"owner":  ownerPerms,
			"admin":  adminPerms,
			"member": memberPerms,
			"viewer": viewerPerms,
		},
	}
}

// HasPermission 检查指定角色是否拥有指定权限。纯函数。
func HasPermission(role, permCode string) bool {
	matrix := DefaultRolePermissionMatrix()
	perms, ok := matrix.Matrix[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permCode {
			return true
		}
	}
	return false
}
