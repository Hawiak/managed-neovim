local M = {}
local util = require("managed.util")
local audit = require("managed.audit")

local BLOCKED_PATTERNS = { ".claude", ".copilot", ".vscode", ".ssh", ".aws" }

local BLOCKED_EVENTS = {
  BufWrite = true, BufWritePre = true, BufWritePost = true,
  FileWritePre = true, FileWritePost = true,
}

local function is_blocked_pattern(pat)
	for _, blocked in ipairs(BLOCKED_PATTERNS) do
		if pat:find(blocked, 1, true) then
			return true
		end
	end
	return false
end

local function check_patterns(patterns)
	local list = type(patterns) == "table" and patterns or { patterns }
	for _, pat in ipairs(list) do
		if type(pat) == "string" and is_blocked_pattern(pat) then
			return pat
		end
	end
end

function M.install()
	local orig = vim.api.nvim_create_autocmd

	--- @diagnostic disable-next-line: duplicate-set-field
	vim.api.nvim_create_autocmd = function(event, opts)
		local plugin = util.calling_plugin()
		if plugin and opts and opts.pattern then
			local events = type(event) == "table" and event or { event }
			local is_write_event = false
			for _, e in ipairs(events) do
				if BLOCKED_EVENTS[e] then is_write_event = true; break end
			end

			local blocked = is_write_event and check_patterns(opts.pattern) or nil
			if blocked then
				local org = vim.env.MANAGED_NVIM_ORG or "managed-nvim"
				audit.log({
					plugin = plugin,
					action = "BLOCKED",
					reason = "autocmd targeting blocked pattern",
					target = blocked,
				})

				error(
					string.format(
						"\n[%s] BLOCKED: %s tried to register an autocmd targeting '%s'.",
						org,
						plugin,
						blocked
					),
					0
				)
			end
		end
		return orig(event, opts)
	end
end

return M
