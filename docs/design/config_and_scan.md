# 配置文件與掃描功能設計文檔 (Design)

## 1. 概述
本設計定義了 Skill Manager 的核心配置格式與掃描機制。透過 JSON 文件管理全局與項目專屬的技能資訊，並利用 MD5 雜湊確保技能的唯一性。

配置目錄預設為 `~/.skill_manager`，但可透過環境變數 `SKILL_MANAGER_CONFIG` 進行自定義。

## 2. 配置文件格式 (JSON)

### 2.1 核心技能配置 (skills.json)
用於記錄所有已掃描並受管理的技能基礎資訊。
- **ID:** `md5(parent_dir/skill_name)`
- **字段說明:**
  - `id`: 唯一識別碼。
  - `name`: 技能名稱（通常為目錄名，亦可由 YAML 指定）。
  - `path`: 技能目錄的絕對路徑。
  - `description`: 技能描述。
  - `metadata`: 字典類型 (Map)，動態保存從 `SKILL.md` 的 YAML Frontmatter 中解析出的其餘字段（如 `scope`, `allowed-tools` 等）。

### 2.2 技能分組配置 (groups.json)
將多個技能打包成組，方便一鍵管理。
- **字段說明:**
  - `id`: 分組 ID。
  - `name`: 分組名稱。
  - `skill_ids`: 所屬技能的 ID 列表。

### 2.3 內容配置 (context.json)
記錄當前工作目錄（或全局環境）下活性的技能與分組。
- **Scope:** `global` 或 `project`。

### 2.4 彙總配置 (summary.json)
記錄所有 `context.json` 的路徑及其更新時間，實現跨專案的快速檢索。

## 3. 核心功能邏輯

### 3.1 技能掃描 (Scanner)
- **識別標誌:** 目錄內是否存在 `SKILL.md`。
- **YAML 解析:** 優先解析 `SKILL.md` 頭部的 YAML Frontmatter (以 `---` 分隔)。
- **Metadata 存儲:** 採用 `yaml:",inline"` 將未在結構體中定義的字段統一存入 `metadata` 字典，方便未來業務擴展。
- **回退機制:** 若無 YAML 頭部，則回退至讀取首行非空文本作為 `description` 的邏輯。
- **ID 生成:** 使用 `crypto/md5` 雜湊 `parent_dir/skill_name`。

### 3.2 CLI 指令
- `init`: 建立配置目錄（預設 `~/.skill_manager`）並初始化基礎 JSON 文件。
- `scan`: 遍歷指定目錄，生成技能列表並與現有 `skills.json` 合併（去重）。
- `list`: 讀取 `skills.json` 並以表格格式輸出。

## 4. 目錄結構
- `main.go`: 進入點（位於根目錄）。
- `internal/cmd/`: Cobra 指令定義。
- `internal/config/`: 配置結構體定義與 I/O 操作。
- `internal/scanner/`: 遍歷、識別與 YAML 解析。
