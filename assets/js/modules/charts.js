import * as echarts from "echarts/core";
// 1. Import komponen PieChart
import { LineChart, PieChart, BarChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  GraphicComponent,
  AxisPointerComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

// 2. Daftarkan di sini
echarts.use([
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  AxisPointerComponent,
  LineChart,
  PieChart,
  BarChart,
  CanvasRenderer,
  GraphicComponent,
]);

export async function getEcharts() {
  return echarts;
}