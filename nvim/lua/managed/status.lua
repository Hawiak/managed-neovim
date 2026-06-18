local M = {}

local function env(key, default)
  return vim.env[key] ~= "" and vim.env[key] or default
end

local function read_manifest()
  local path = vim.env.MANAGED_NVIM_MANIFEST_PATH
  if not path or path == "" then return nil end
  local ok, data = pcall(vim.fn.readfile, path)
  if not ok then return nil end
  local ok2, decoded = pcall(vim.fn.json_decode, table.concat(data, "\n"))
  if not ok2 then return nil end
  return decoded
end

local function build_lines(manifest)
  local org      = env("MANAGED_NVIM_ORG", "Unknown")
  local sandbox  = vim.env.MANAGED_NVIM_SANDBOX == "1"
  local date     = env("MANAGED_NVIM_MANIFEST_DATE", "unknown")
  local count    = env("MANAGED_NVIM_PLUGIN_COUNT", "?")
  local schema   = env("MANAGED_NVIM_SCHEMA_VERSION", "?")
  local wrapper  = vim.env.MANAGED_NVIM == "1"

  local v = vim.version()
  local nvim_ver = string.format("NVIM v%d.%d.%d", v.major, v.minor, v.patch)

  local width = 54
  local title = string.format("managed-nvim  ·  %s", org)
  local pad = math.floor((width - #title) / 2)

  local lines = {
    "",
    string.rep(" ", pad) .. title,
    "",
    string.rep("─", width),
    "",
    "  Security",
    "  " .. string.rep("─", width - 2),
    string.format("  %-22s %s", "Sandbox",
      sandbox and "● running (macOS sandbox-exec)" or "○ not active"),
    string.format("  %-22s %s", "File locks",
      wrapper and "● enabled (chflags uchg)" or "○ not active"),
    string.format("  %-22s %s", "Wrapper",
      wrapper and "● active" or "○ not running — launch via nvim-wrapper"),
    "",
    "  Manifest",
    "  " .. string.rep("─", width - 2),
    string.format("  %-22s %s", "Organisation",  org),
    string.format("  %-22s %s", "Last updated",  date),
    string.format("  %-22s %s plugins approved", "Plugins", count),
    string.format("  %-22s %s", "Schema version", schema),
    "",
    "  Runtime",
    "  " .. string.rep("─", width - 2),
    string.format("  %-22s %s", "Neovim", nvim_ver),
    string.format("  %-22s %s", "Manifest path",
      vim.env.MANAGED_NVIM_MANIFEST_PATH or "not set"),
    "",
    string.rep("─", width),
    "",
  }

  return lines
end

local function set_highlights(buf)
  vim.api.nvim_set_hl(0, "ManagedNvimTitle",   { bold = true, fg = "#89b4fa" })
  vim.api.nvim_set_hl(0, "ManagedNvimSection", { bold = true, fg = "#cdd6f4" })
  vim.api.nvim_set_hl(0, "ManagedNvimRule",    { fg = "#45475a" })
  vim.api.nvim_set_hl(0, "ManagedNvimKey",     { fg = "#a6adc8" })
  vim.api.nvim_set_hl(0, "ManagedNvimOk",      { fg = "#a6e3a1" })
  vim.api.nvim_set_hl(0, "ManagedNvimWarn",    { fg = "#f9e2af" })

  local function match(pattern, group, priority)
    vim.fn.matchadd(group, pattern, priority or 10)
  end

  vim.api.nvim_buf_call(buf, function()
    match([[managed-nvim  ·  .*]], "ManagedNvimTitle", 20)
    match([[●.*]],                 "ManagedNvimOk")
    match([[○.*]],                 "ManagedNvimWarn")
    match([[  Security]],          "ManagedNvimSection")
    match([[  Manifest]],          "ManagedNvimSection")
    match([[  Runtime]],           "ManagedNvimSection")
    match([[─\+]],                 "ManagedNvimRule")
  end)
end

function M.open()
  -- Reuse existing buffer if already open
  for _, b in ipairs(vim.api.nvim_list_bufs()) do
    if vim.api.nvim_buf_get_name(b) == "managed-nvim://status" then
      vim.api.nvim_set_current_buf(b)
      return
    end
  end

  local manifest = read_manifest()
  local lines = build_lines(manifest)

  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_name(buf, "managed-nvim://status")
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)

  vim.bo[buf].buftype   = "nofile"
  vim.bo[buf].bufhidden = "wipe"
  vim.bo[buf].swapfile  = false
  vim.bo[buf].modifiable = false
  vim.bo[buf].filetype  = "managed-nvim"

  vim.api.nvim_set_current_buf(buf)
  set_highlights(buf)

  -- q closes the buffer
  vim.keymap.set("n", "q", "<cmd>bwipeout<cr>", { buffer = buf, silent = true })
end

return M
