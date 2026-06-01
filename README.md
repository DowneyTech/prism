# Prism

週次レポートを起点に、AIがプロジェクト全体の状態を可視化するチーム向けSaaS。

## Features

- 週次レポートの入力・管理
- AI による週次サマリー自動生成（BYOK：各チームが自分の AI API キーを使用）
- ダッシュボードでチームの提出状況・達成度を可視化
- Todoist 連携によるレポート入力の自動補完
- リマインダーメール通知

## Tech Stack

**Frontend**
- Next.js 14 (App Router) / TypeScript
- Tailwind CSS + shadcn/ui
- Recharts

**Backend**
- Go / Echo
- PostgreSQL + sqlc
- JWT 認証 + Google OAuth2

**Infrastructure**
- Docker Compose
- Nginx (reverse proxy)
- GitHub Actions (CI/CD → VPS deploy)

## Getting Started

```bash
# 環境変数の設定
cp .env.example .env

# 起動
docker compose up
```

- Frontend: http://localhost:3000
- Backend: http://localhost:8080

## License

MIT
