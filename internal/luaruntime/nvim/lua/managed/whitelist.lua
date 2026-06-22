local M = {}

local function load_approved()
	local path = vim.env.MANAGED_NVIM_MANIFEST_PATH
	if not path or path == "" then
		return nil
	end

	-- Load the manifest file if it exists as lines
	local ok, lines = pcall(vim.fn.readfile, path)
	if not ok then
		return nil
	end

	-- Actually try to parse the manifest
	local ok2, manifest = pcall(vim.fn.json_decode, table.concat(lines, "\n"))
	if not ok2 or not manifest.plugins then
		return nil
	end

	-- Iterate over the plugins and add them to the approved list
	local approved = {}
	for _, plugin in ipairs(manifest.plugins) do
		approved[plugin.repo:lower()] = true
	end
	return approved
end

-- Figure out the repo string, skip local plugins, and return nil for them
local function extract_repo(spec)
	if type(spec) == "string" then
		return spec
	end

	if type(spec) == "table" then
		if spec.dir then
			return nil
		end --local plugin, skip
		if type(spec[1]) == "string" then
			return spec[1]
		end
	end
	return nil
end

local function check_plugins(plugins, approved)
	for _, spec in ipairs(plugins) do
		if type(spec) == "table" and not spec[1] and not spec.dir then
			-- Nested group, check recursively
			check_plugins(spec, approved)
		else
			local repo = extract_repo(spec)
			local org = vim.env.MANAGED_NVIM_ORG or "managed-nvim"
			if repo and not approved[repo:lower()] then
				error(
					string.format(
						"\n[%s] BLOCKED: %s is not whitelisted!\nContact your administrator to request approval.",
						org,
						repo
					),
					0 -- Silence lua blabla
				)
				-- TODO: It would be nice if we can print the plugin specs here and have a configurable
				-- Option to create a servicenow ticket??
			end
		end
	end
end

function M.install()
	local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
	if not vim.loop.fs_stat(lazypath) then
		return -- not installed yet, skip wrapping
	end

	vim.opt.rtp:prepend(lazypath)
	local lazy = require("lazy")
	local orig = lazy.setup -- copy the original setup function
	lazy.setup = function(plugins, opts)
		local approved = load_approved()
		if approved then
			check_plugins(plugins, approved)
		end
		return orig(plugins, opts) -- call the original setup function
	end

	return lazy
end

return M
