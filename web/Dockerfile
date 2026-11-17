# --- build the React app ---
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

# --- serve it with Nginx ---
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template

ENV MENU_API_URL=http://menu-api:8000 \
    ORDERS_API_URL=http://orders-api:8080 \
    STATS_API_URL=http://stats-api:8001

EXPOSE 80
