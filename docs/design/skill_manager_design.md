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

### 2.2 技能組 (groups.json)
儲存使用者定義的技能包 (Skill Packs)。
- **ID:** 使用分組名稱作為 ID。
- **核心字段:**
  - `id`: 唯一識別碼。
  - `name`: 分組名稱。
  - `description`: 分組描述。
  - `skill_ids`: 字串列表，記錄該分組包含的所有技能 ID。

### 2.3 上下文配置 (.context.json / global_context.json)
記錄當前專案或全局環境下的技能使用狀態。
- **技能使用狀況 (SkillUsage):**
  - `id`: 技能 ID。
  - `agents`: 字串列表，記錄該技能被軟鏈到了哪些 Agent（如 `["gemini", "claude"]`）。
- **ContextConfig:**
  - `scope`: `global` 或 `project`。
  - `active_skills`: `[]SkillUsage`，記錄目前啟用的單體技能及其關聯的 Agent。
  - `inactive_skills`: `[]SkillUsage`，記錄目前已停用但保留在專案配置中的技能。
  - `active_groups`: `[]string`，記錄目前啟用的技能組名稱清單。

## 3. 功能邏輯

### 3.1 技能掃描 (Scanner)
...

### 3.2 技能生命週期管理
- **Add:**
  - 支援 `--group <name>`：將分組加入 `active_groups`。
  - 支援單體技能：將技能加入 `active_skills`。
  - 若指定多個 Agent，會合併現有的 `agents` 名單。
  - 自動觸發 **Sync** 邏輯。
- **Stop:**
  - 根據 `--agent` 指定移除特定 Agent 下的軟鏈。
  - 更新 `active_skills` 中的 `agents` 名單。
- **Remove:**
  - 支援 `--group <name>`：從 `active_groups` 中移除分組。
  - 支援單體技能：從 `active_skills` 完全刪除。
  - 自動觸發 **Sync** 邏輯。
- **Sync:**
  - 計算預期狀態：結合 `active_skills` 與 `active_groups` 內的所有技能，決定每個 Agent 目錄下應該存在的軟鏈。
  - 套用變更：建立缺失的軟鏈，並移除多餘的「孤兒」軟鏈。
  - **衝突處理:** 若目標位置已存在非對應的檔案或連結，且未提供 `--force` 參數，則中止操作並提示衝突。
- **Status:**
  - 展示當前上下文的所有活性技能、啟用的分組及停用技能。

## 4. 命令結構 (CLI)
- **group:** 管理技能組（create, delete, add-skill, remove-skill, list）。
- **add / remove:** 支援單體技能或使用 `--group` 標誌。
- **sync:** 強制同步專案環境與配置狀態。
- **scan:** 更新全局技能庫。

## 4. 目錄與配置
- **全局配置:** 預設為 `~/.skill_manager`。
- **環境變數:** `SKILL_MANAGER_CONFIG` 可自定義全局配置路徑。
- **Agent 目錄:** 支援 `.gemini`, `.claude`, `.codex`, `.opencode`, `.agents` 等。
