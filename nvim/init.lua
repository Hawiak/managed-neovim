-- managed-nvim entry point
-- This file is loaded by the wrapper in place of the user's init.lua.
-- It sets up the managed layer first, then loads the user's own config.

require("managed").setup()
