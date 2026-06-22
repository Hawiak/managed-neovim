local M = {}
local util = require("managed.util")
local audit = require("managed.audit")

local function required_permission(cmd)
	local bin = type(cmd) == "table" and cmd[1] or cmd
	bin = bin:match("([^/]+)$") or bin

	local map = {
		git = "git",
		stylua = "format",
		prettier = "format",
		eslint = "format",
		rg = "format",
		fd = "format",
		fzf = "format",
		make = "build",
		cargo = "build",
		cmake = "build",
		cc = "build",
		gcc = "build",
		go = "build",
		npm = "install",
		pip = "install",
		pip3 = "install",
		gem = "install",
		mason = "install",
		bash = "shell",
		sh = "shell",
		zsh = "shell",
		fish = "shell",
	}

	return map[bin] or "shell"
end

function M.install()
	local permissions = util.load_permissions()
	if not permissions then
		return
	end

	local orig_jobstart = vim.fn.jobstart
	local orig_system = vim.fn.system

	-- Override jobstart to check permissions for plugins trying to spawn processes
	-- jobstart is used by many plugins to run external commands asynchronously, so we need to check permissions here as well
	--- @diagnostic disable-next-line: duplicate-set-field
	vim.fn.jobstart = function(cmd, ...)
		local plugin = util.calling_plugin()
		if plugin then
			local required = required_permission(cmd)
			local org = vim.env.MANAGED_NVIM_ORG or "managed-nvim"
			if not (permissions[plugin] and permissions[plugin][required]) then
				audit.log({
					plugin = plugin,
					action = "BLOCKED",
					reason = "lacks '" .. required .. "' permissions",
					target = type(cmd) == "table" and cmd[1] or cmd,
				})
				error(
					string.format(
						"\n[%s] BLOCKED: %s tried to spawn a process but lacks the '%s' permission",
						org,
						plugin,
						required
					),
					0
				)
			end
		end
		return orig_jobstart(cmd, ...)
	end

	-- Override the system function to check permissions for plugins trying to execute shell commands
	--- @diagnostic disable-next-line: duplicate-set-field
	vim.fn.system = function(cmd, ...)
		local plugin = util.calling_plugin()
		if plugin then
			local required = required_permission(cmd)
			local org = vim.env.MANAGED_NVIM_ORG or "managed-nvim"
			if not (permissions[plugin] and permissions[plugin][required]) then
				audit.log({
					plugin = plugin,
					action = "BLOCKED",
					reason = "lacks '" .. required .. "' permissions",
					target = type(cmd) == "table" and cmd[1] or cmd,
				})
				error(
					string.format(
						"\n[%s] BLOCKED: %s tried to spawn a process but lacks the %s permission",
						org,
						plugin,
						required
					),
					0
				)
			end
		end
		return orig_system(cmd, ...)
	end
end

return M
