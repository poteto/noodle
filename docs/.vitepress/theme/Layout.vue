<script setup>
import DefaultTheme from "vitepress/theme";
import { useData } from "vitepress";
import HomePage from "./HomePage.vue";
import NotFound from "./NotFound.vue";

const { frontmatter, page } = useData();
</script>

<template>
  <HomePage v-if="frontmatter.customHome" />
  <DefaultTheme.Layout v-else>
    <template #not-found>
      <NotFound />
    </template>
  </DefaultTheme.Layout>
</template>
