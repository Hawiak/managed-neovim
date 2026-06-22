local M = {}

function M.setup()
	-- Before we do anything, check if the user has started the NVIM through the wrapper or not
	if not vim.env.MANAGED_NVIM or vim.env.MANAGED_NVIM == "" then
		error(
			"\n[managed-nvim] This Neovim instance was not launched through the managed wrapper.\nYour company requires you to start it through the wrapper, contact your admin for instructions",
			0
		)
	end
	-- Must run before lazy.setup() fires in the user's config
	require("managed.whitelist").install()
	require("managed.hooks_blocker").install()
	require("managed.network_shim").install()
	-- Load before lazy.nvim replaces the require loader
	local status = require("managed.status")

	vim.api.nvim_create_user_command("ManagedNeovim", function()
		status.open()
	end, { desc = "Open managed-nvim status page" })
end

return M
