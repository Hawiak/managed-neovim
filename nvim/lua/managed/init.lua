local M = {}

function M.setup()
  -- Must run before lazy.setup() fires in the user's config
  require("managed.whitelist").install()

  -- Load before lazy.nvim replaces the require loader
  local status = require("managed.status")

  vim.api.nvim_create_user_command("ManagedNeovim", function()
    status.open()
  end, { desc = "Open managed-nvim status page" })
end

return M
