# Crescendo Frontend

A modern React frontend for the Crescendo music downloader application.

## Features

- 🎵 Search music tracks, albums, and artists
- ⬇️ Download queue management with real-time progress
- 📁 File browser for downloaded music
- 🎨 Modern UI with shadcn/ui components
- 🌙 Dark/light mode support
- 📱 Mobile responsive design

## Tech Stack

- **React** with TypeScript
- **Vite** for development and building
- **shadcn/ui** for UI components
- **Tailwind CSS** for styling
- **React Router** for navigation
- **Lucide React** for icons

## Development

### Prerequisites

- Node.js 18+ and npm
- Crescendo backend running on localhost:8080

### Getting Started

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the development server:
   ```bash
   npm run dev
   ```

3. Open http://localhost:5173 in your browser

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## Backend Integration

The frontend connects to the Crescendo backend running on localhost:8080. Make sure the backend is running before using the application.

### API Endpoints Used

- `GET /health` - Connection health check
- `GET /api/search` - Search for music
- `POST /api/downloads/*` - Queue downloads
- `GET /api/downloads` - Get download status
- `GET /api/files` - Browse downloaded files
- `WebSocket /ws/downloads/*` - Real-time progress updates

## Project Structure

```
src/
├── components/          # React components
│   ├── ui/             # shadcn/ui components
│   ├── layout/         # Layout components
│   ├── search/         # Search page components
│   ├── downloads/      # Downloads page components
│   └── files/          # Files page components
├── hooks/              # Custom React hooks
├── services/           # API integration
├── types/              # TypeScript type definitions
└── utils/              # Helper functions
```

## Connection Status

The application shows connection status in the header:
- 🟢 Connected - Backend is reachable
- 🔴 Disconnected - Backend is not available

## Development Notes

This is the first slice of the Crescendo frontend implementation, providing:
- ✅ Complete project setup and build configuration
- ✅ Modern layout with navigation
- ✅ Backend connectivity and health checks
- ✅ Placeholder pages for all main features
- ✅ Type-safe API integration layer

Next development phases will add:
- Search functionality with results display
- Download queue management
- Real-time progress updates
- File browser with audio playback