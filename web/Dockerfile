# Фронт: сборка Vite и раздача через nginx, который проксирует /api на Go-сервис.
# Отдельная цель sim — генератор парка для демо-стенда (docker compose --profile demo)

FROM node:24-alpine AS deps
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci

FROM deps AS build
COPY . .
# учётка демо-стенда попадает в сборку как публичные данные витрины; без неё кнопки демо нет
ARG VITE_DEMO_EMAIL=""
ARG VITE_DEMO_PASSWORD=""
ENV VITE_FLEET_SOURCE=api \
    VITE_DEMO_EMAIL=$VITE_DEMO_EMAIL \
    VITE_DEMO_PASSWORD=$VITE_DEMO_PASSWORD
RUN npm run build

FROM deps AS sim
COPY . .
USER node
CMD ["npm", "run", "sim"]

FROM nginx:1.29-alpine AS web
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /web/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
