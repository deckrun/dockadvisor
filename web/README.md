# Dockadvisor Static Build

This is a standalone static version of Dockadvisor - a free online Dockerfile analyzer and linter.

## Overview

This build uses **Vite + React + TailwindCSS** to create static HTML/CSS/JS files that can be served by nginx or any static file server, without requiring Node.js at runtime.

## Features

- **Monaco Editor** integration for Dockerfile editing
- **WebAssembly-based** parsing and validation (runs entirely client-side)
- **50+ validation rules** across 18 Dockerfile instructions
- **Real-time feedback** on best practices, security, and optimization
- **Share links** with compressed Dockerfile content
- **Example templates** for learning

## Prerequisites

- Node.js 18+ (for building only, not for serving)
- npm or yarn

## Installation

```bash
npm install
```

## Development

Run the development server:

```bash
npm run dev
```

This will start Vite's dev server at `http://localhost:3000`

## Building for Production

Create a production build:

```bash
npm run build
```

This generates static files in the `dist/` directory.

## Preview Production Build

Test the production build locally:

```bash
npm run preview
```

## Deployment

### Nginx

Copy the `dist/` folder to your web server and configure nginx:

```nginx
server {
    listen 80;
    server_name dockadvisor.example.com;
    root /path/to/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### Apache

```apache
<VirtualHost *:80>
    ServerName dockadvisor.example.com
    DocumentRoot /path/to/dist

    <Directory /path/to/dist>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted

        # Enable client-side routing
        RewriteEngine On
        RewriteBase /
        RewriteRule ^index\.html$ - [L]
        RewriteCond %{REQUEST_FILENAME} !-f
        RewriteCond %{REQUEST_FILENAME} !-d
        RewriteRule . /index.html [L]
    </Directory>
</VirtualHost>
```

### Static File Hosting Services

The `dist/` folder can be deployed to:
- GitHub Pages
- Netlify
- Vercel
- Cloudflare Pages
- AWS S3 + CloudFront
- Any static hosting service

## Project Structure

```
static/
├── public/
│   └── js/
│       ├── wasm_exec.js       # Go WASM runtime
│       └── dockadvisor.wasm   # Dockerfile parser
├── src/
│   ├── components/
│   │   ├── Container.jsx
│   │   ├── Logo.jsx
│   │   ├── Header.jsx
│   │   ├── DockadvisorContext.jsx
│   │   ├── DockadvisorClient.jsx
│   │   ├── Dockadvisor.jsx
│   │   └── DockadvisorPage.jsx
│   ├── styles/
│   │   └── tailwind.css
│   ├── main.jsx
│   └── App.jsx
├── fonts/
│   └── lexend.woff2
├── index.html
├── package.json
├── vite.config.js
└── postcss.config.js
```

## Key Dependencies

- **React 19** - UI library
- **Monaco Editor** - Code editor
- **TailwindCSS v4** - CSS framework
- **HeadlessUI** - Accessible UI components
- **LZ-String** - Compression for share links
- **WebAssembly** - Client-side Dockerfile parsing

## Privacy

All Dockerfile analysis happens **entirely in your browser** using WebAssembly. No data is sent to any server. This makes it safe to analyze proprietary and sensitive Dockerfiles.

## License

See the main Deckrun project for licensing information.

## Attribution

Dockadvisor uses open source software including [moby/buildkit](https://github.com/moby/buildkit) licensed under the Apache License 2.0.
