<template>
  <apexcharts
    width="100%"
    height="400"
    type="bar"
    :options="options"
    :series="series"
  />
</template>

<script>
import VueApexCharts from "vue-apexcharts";

export default {
  name: "barGraph",
  components: {
    apexcharts: VueApexCharts,
  },
  props: {
    options: {
      type: Object,
      default: null,
    },
    series: {
      type: Array,
      default: null,
    },
  },
};
</script>
