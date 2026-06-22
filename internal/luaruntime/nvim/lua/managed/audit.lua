local M = {}

local function log_path()
	return vim.env.MANAGED_NVIM_AUDIT_LOG or vim.fn.stdpath("data") .. "/managed-nvim-audit.log"
end

function M.tail(numberOfLines)
	local ok, lines = pcall(vim.fn.readfile, log_path())
	if not ok then
		return nil
	end

	local slice = vim.list_slice(lines, math.max(1, #lines - numberOfLines + 1), #lines)

	local result = {}
	for _, line in ipairs(slice) do
		local ok2, decoded = pcall(vim.fn.json_decode, line)
		if ok2 then
			table.insert(result, decoded)
		end
	end

	return result
end

function M.log(event)
	if type(event) ~= "table" then
		error("Event must be a table")
	end

	event.timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ")
	local serialized = vim.fn.json_encode(event)
	local path = log_path()
	vim.fn.writefile({ serialized }, path, "a")
end

return M
