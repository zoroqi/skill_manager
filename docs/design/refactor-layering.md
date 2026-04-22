# 代碼分層重構設計文檔 (Code Layering Refactor)

## 1. 背景與問題

目前 Skill Manager 的代碼結構存在職責邊界模糊的問題，主要體現在：
- **CLI 層過重**：`cmd/` 目錄除了處理參數解析外，還包含了大量的業務邏輯（如按名查找技能、判斷分組邏輯、合併 Agent 列表等）。
- **Manager 職責混合**：目前的 `config.Manager` 既負責底層的文件 I/O，又承載了業務邏輯（如 `AddSkillToContext`、`SyncContext`、`StopSkillInContext`）。
- **代碼復用性差**：當未來引入 TUI（終端用戶介面）時，必須在 TUI 層重新實現 `cmd/` 中的業務邏輯，導致重複代碼。

## 2. 新分層結構

重構後的目錄結構將明確區分「類型定義」、「業務邏輯」與「數據持久化」：

```
internal/
  ├── types.go        ← 所有共享類型（Skill, Group, ContextConfig 等）
  ├── app/            ← Use Cases：核心業務邏輯，CLI/TUI 共用
  │     ├── app.go        (App 結構體定義與初始化)
  │     ├── skill.go      (AddSkill, RemoveSkill, StopSkill, ListSkills)
  │     ├── group.go      (CreateGroup, DeleteGroup, AddSkillToGroup, RemoveSkillFromGroup, ListGroups)
  │     ├── context.go    (SyncContext, GetStatus)
  │     ├── scan.go       (ScanSkills)
  │     ├── init.go       (Initialize)
  │     └── util.go       (私有輔助函數，如 mergeAgents)
  ├── store/          ← 持久化層：純粹的文件 I/O 操作（原 config/ 移除業務方法後重命名）
  │     ├── manager.go    (Load*/Save* 系列底層操作)
  │     ├── agent.go      (FindExistingAgents, EnsureAgentSkillsDir, RemoveAgentSymlinks)
  │     └── context.go    (EnsureLocalContext, SaveLocalContext)
  ├── scanner/        ← 技能掃描邏輯（保持現狀）
  │     └── scanner.go
  ├── cmd/            ← 展示層 (CLI)：僅負責解析參數、調用 app 層並格式化輸出
  │     └── *.go
  └── tui/            ← 展示層 (TUI)：未來擴充（當前不在本次重構範圍內）
```

## 3. 各層職責定義

| 層級 | 職責說明 |
| :--- | :--- |
| **types** | 定義系統核心領域模型與共享結構體。不包含任何邏輯。 |
| **app (Use Cases)** | 實現業務用例。處理複雜邏輯、狀態校驗、數據組合。是系統的靈魂。 |
| **store (Storage)** | 負責數據的讀取與寫入。屏蔽文件系統、JSON 序列化等細節。 |
| **cmd / tui** | 負責與用戶交互。參數驗證、調用 app 層、渲染輸出結果。 |
| **scanner** | 專注於從文件系統解析技能元數據。 |

## 4. 依賴方向規則

為確保代碼的低耦合性，強制執行單向依賴規則：

```
cmd / tui  ──→  app  ──→  store / scanner
    ↑            ↑
    └────────────┴────── types (所有層級均可引用)
```

**硬性限制：**
- `cmd` 與 `tui` **禁止**直接引用 `store`。所有數據操作必須通過 `app` 層。
- `store` 與 `scanner` **禁止**引用 `app` 層。
- `types.go` 作為純類型定義，**禁止**依賴任何其他內核模塊。

## 5. 遷移步驟

重構將分為三個階段進行，以確保每一步都可測試：

### 第一步：抽取類型定義
- 將 `config/config.go` 中的核心結構體移動至 `internal/types.go`。
- 更新全域 import，確保編譯通過。

### 第二步：建立純淨持久化層 (Store)
- 將 `config/` 目錄重命名為 `internal/store/`。
- 移除其中所有涉及業務決策的方法（如 `AddSkillToContext`）。
- 僅保留底層的 `Load`、`Save` 與文件系統操作函數。

### 第三步：建立業務邏輯層 (App)
- 創建 `internal/app/` 目錄。
- 將原先散落在 `cmd/` 和原 `config.Manager` 中的業務邏輯收攏至 `app` 層。
- 為 `app` 層編寫 TDD 測試。
- 修改 `cmd/` 代碼，將其簡化為調用 `app` 實例的方法。
