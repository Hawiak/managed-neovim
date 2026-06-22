-- managed-nvim entry point
-- Loaded via -u flag so the managed security layer runs before any user config.
-- Neovim still adds ~/.config/nvim to runtimepath with -u, but skips sourcing
-- the user's init — we do that explicitly at the end.

require("managed").setup()

local user_init = vim.fn.stdpath("config") .. "/init.lua"
if vim.fn.filereadable(user_init) == 1 then
	local ok, err = pcall(dofile, user_init)
	if not ok then
		vim.notify("[managed-nvim] Error loading user config: " .. tostring(err), vim.log.levels.ERROR)
	end
else
	local user_init_vim = vim.fn.stdpath("config") .. "/init.vim"
	if vim.fn.filereadable(user_init_vim) == 1 then
		vim.cmd("source " .. vim.fn.fnameescape(user_init_vim))
	end
end
