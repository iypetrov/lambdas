/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.templ",
    "./static/js/**/*.js"
  ],
  safelist: ['alert', 'alert-success', 'alert-error'],
  theme: {
    extend: {
      fontFamily: {
        primary: ['Mukta', 'sans-serif'],
        accent: ['Lacquer', 'sans-serif'],
      },
    },
  },
  plugins: [],
};
