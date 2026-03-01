<script setup>
import DefaultTheme from "vitepress/theme";
import { useData } from "vitepress";
import HomePage from "./HomePage.vue";

const { frontmatter } = useData();
</script>

<template>
  <HomePage v-if="frontmatter.customHome" />
  <DefaultTheme.Layout v-else />
</template>
