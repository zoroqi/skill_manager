# Skill Manager 核心設計文檔 (Core Design)

## 1. 概述
Skill Manager 是一款專為 AI Agent 設計的本地技能管理工具。透過掃描特定目錄下的技能資訊，建立全局的技能庫，並動態將技能軟鏈 (Symbolic Link) 到各類 Agent 的特定目錄中（如 `.gemini/skills`, `.claude/skills`）。

## 2. 數據模型 (Data Model)

### 2.1 技能庫 (skills.json)
儲存所有已掃描並受管理的技能。
- **ID:** 使用 `md5(registry_name_or_parent_dir_skill_name)` 以確保唯一性。
- **核心字段:**
  - `id`: 唯一識別碼。
  - `name`: 技能名稱（通常為目錄名，亦可由 YAML 指定）。
  - `registry_name`: 所屬註冊專案名稱（若為本地掃描則為空）。
  - `path`: 技能目錄的路徑。對於註冊的技能，存儲相對於註冊庫根目錄的**相對路徑**；在內存中加載時還原為絕對路徑。
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

### 2.3 遠端倉庫註冊 (registries.json)
記錄透過 `register` 功能從開源平臺下載的技能專案。
- **核心字段:**
  - `name`: 註冊名稱，採用 `<org>_<repo>` 格式以避免命名衝突。
  - `url`: 原始 Git URL (支援 HTTPS/SSH)。
  - `local_path`: 下載後的本地邏輯路徑，預設包含 `$SKILL_MANAGER_REGISTER_PATH` 前綴。
  - `last_updated`: 最後一次 `update` 的時間。

### 2.4 上下文配置 (.context.json / global_context.json)
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
遍歷目標目錄及其子目錄，尋找包含 `SKILL.md` 的資料夾。此功能主要由 `register` 流程內部觸發。
- **遞歸掃描:** 使用 `filepath.WalkDir` 支援發現深層結構（如 `skills/` 子目錄）中的技能。
- **安全過濾:** 自動跳過隱藏目錄（如 `.git/`, `.github/`）。
- **原子性發現:** 一旦在某個目錄發現 `SKILL.md`，即將該目錄視為一個技能單元，停止向該目錄深層繼續掃描，避免重複識別。

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
- **Register (Git-centric):**
  - **Add:** 執行 `git clone`，將倉庫存放到預設路徑（環境變數或 `registries/` 目錄），並自動觸發 `ScanSkills`。在 `registries.json` 中使用邏輯路徑存儲，以便遷移。
  - **Update:** 針對註冊專案執行 `git pull` 並重新執行 `ScanSkills`。
  - **Remove:** 支援 `--delete` 參數，將本地目錄移動至備份區，並清理 `skills.json` 中的無效路徑。

## 4. 命令結構 (CLI)
- **register:** 管理遠端技能倉庫 (add, update, list, remove)。
- **group:** 管理技能組（create, delete, add-skill, remove-skill, list）。
- **add / remove:** 支援單體技能或使用 `--group` 標誌。
- **sync:** 強制同步專案環境與配置狀態。
- **status:** 展示當前上下文的所有活性技能、啟用的分組及停用技能。

## 5. 目錄與配置
- **全局配置目錄:** 預設為 `~/.skill_manager`。
- **環境變數:**
  - `SKILL_MANAGER_CONFIG`: 自定義全局配置路徑（包含 `skills.json`, `registries.json` 等）。
  - `SKILL_MANAGER_REGISTER_PATH`: 自定義遠端技能倉庫的基礎存放路徑。
- **Agent 目錄:** 支援 `.gemini`, `.claude`, `.codex`, `.opencode`, `.agents` 等。
