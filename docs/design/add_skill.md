# Skill Management (Add) 功能設計文檔 (Design)

## 1. 概述
本設計定義了 Skill Manager 的 `add` 命令邏輯。用於將已掃描的技能添加到當前專案或全局環境中，並建立與各種 Agent（如 Gemini, Claude）的關聯。

## 2. 軟鏈機制 (Symlinking)
當執行 `add` 時，系統會在目標 Agent 專用的隱藏目錄下建立 `skills` 目錄，並將源技能目錄軟鏈接 (Symbolic Link) 過去。
- **目標 Agent 目錄:** 由 \`--agent\` 參數指定（如 \`gemini\` 對應 \`.gemini/skills\`，\`claude\` 對應 \`.claude/skills\` 等）。
- 若目錄不存在，根據用戶要求，系統會主動創建指定的目標目錄及其 \`skills\` 子目錄。

## 3. 上下文配置 (Context Config)
- **Global (\`--global\`):** 更新 \`~/.skill_manager/global_context.json\`，在 \`active_skills\` 中新增技能。
- **Project (Local):** 預設情況下，會在當前工作目錄建立並更新 \`.context.json\`，在 \`active_skills\` 中新增技能。

## 4. CLI 命令
\`skill_manager add <skill_name> [--global] [--agent <agent_names>]\`
- \`<skill_name>\`: 技能名稱。
- \`--global\`: 指定為全局作用域。
- \`--agent\`: 指定要綁定並創建的 Agent 目錄，支援多個值（例如 \`gemini,claude\`）。若未指定，則嘗試尋找已存在的 Agent 目錄，若都無則報錯並提示需指定 \`--agent\`。
