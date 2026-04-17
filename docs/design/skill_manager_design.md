# Skill Manager 核心設計文檔 (Core Design)

## 1. 概述
Skill Manager 是一款專為 AI Agent 設計的本地技能管理工具。透過掃描特定目錄下的技能資訊，建立全局的技能庫，並動態將技能軟鏈 (Symbolic Link) 到各類 Agent 的特定目錄中（如 `.gemini/skills`, `.claude/skills`）。

## 2. 數據模型 (Data Model)

### 2.1 技能庫 (skills.json)
儲存所有已掃描並受管理的技能。
- **ID:** 使用 `md5(parent_dir/skill_name)` 以確保唯一性。
- **核心字段:**
  - `id`: 唯一識別碼。
  - `name`: 技能名稱（通常為目錄名，亦可由 YAML 指定）。
  - `path`: 技能目錄的絕對路徑。
  - `description`: 技能描述。
  - `metadata`: `map[string]any`，用於存儲從 `SKILL.md` 中解析出的 YAML 前言資訊。

### 2.2 上下文配置 (.context.json / global_context.json)
記錄當前專案或全局環境下的技能使用狀態。
- **技能使用狀況 (SkillUsage):**
  - `id`: 技能 ID。
  - `agents`: 字串列表，記錄該技能被軟鏈到了哪些 Agent（如 `["gemini", "claude"]`）。
- **ContextConfig:**
  - `scope`: `global` 或 `project`。
  - `active_skills`: `[]SkillUsage`，記錄目前啟用的技能及其關聯的 Agent。
  - `inactive_skills`: `[]SkillUsage`，記錄目前已停用但保留在專案配置中的技能。

## 3. 功能邏輯

### 3.1 技能掃描 (Scanner)
- 透過偵測目錄內是否存在 `SKILL.md` 來識別技能。
- 支援解析 `SKILL.md` 頭部的 YAML Frontmatter。

### 3.2 技能生命週期管理
- **Add:**
  - 將技能加入 `active_skills`。
  - 若指定多個 Agent，會合併現有的 `agents` 名單。
  - 自動從 `inactive_skills` 移除該技能（若存在）。
  - 在指定的 Agent 目錄（如 `.gemini/skills/`）建立軟鏈。
- **Stop:**
  - 根據 `--agent` 指定移除特定 Agent 下的軟鏈。
  - 更新 `active_skills` 中的 `agents` 名單。
  - 當技能不再有關聯的 Agent 時，將其移至 `inactive_skills`。
- **Remove:**
  - 徹底移除該技能的所有軟鏈。
  - 從 `active_skills` 和 `inactive_skills` 中完全刪除該技能。
- **Status:**
  - 展示當前上下文的所有活性與停用技能，並列出每個技能關聯的 Agent 清單。

## 4. 目錄與配置
- **全局配置:** 預設為 `~/.skill_manager`。
- **環境變數:** `SKILL_MANAGER_CONFIG` 可自定義全局配置路徑。
- **Agent 目錄:** 支援 `.gemini`, `.claude`, `.codex`, `.opencode`, `.agents` 等。
