# --- Étape 1 : Build ---
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# --- Étape 2 : Production ---
FROM node:22-alpine AS runner
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./

RUN npm ci --only=production

RUN npm cache clean --force

EXPOSE 3000
CMD ["node", "dist/main"]