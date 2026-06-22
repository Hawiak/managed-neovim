local M = {}

function M.calling_plugin()
	local level = 2
	while true do
		local info = debug.getinfo(level, "S")
		if not info then
			return nil
		end

		local source = info.source or ""
		local name = source:match("/lazy/([^/]+)/")
		if name then
			return name:lower()
		end
		level = level + 1
	end
end

function M.load_permissions()
	local raw = vim.env.MANAGED_NVIM_PERMISSIONS
	if not raw or raw == "" then
		return nil
	end

	local ok, data = pcall(vim.fn.json_decode, raw)
	if not ok or type(data) ~= "table" then
		return nil
	end

	local permissions = {}
	for name, list in pairs(data) do
		local entry = {}
		for _, p in ipairs(list) do
			entry[p] = true
		end
		permissions[name:lower()] = entry
	end
	return permissions
end

return M
