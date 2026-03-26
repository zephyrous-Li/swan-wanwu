import service from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

/**
 * 模型统计接口
 */

// 获取模型统计数据
export const getModelData = params => {
  return service({
    url: `${USER_API}/statistic/model`,
    method: 'get',
    params,
  });
};

// 获取模型列表
export const fetchModelList = params => {
  return service({
    url: `${USER_API}/statistic/model/list`,
    method: 'get',
    params,
  });
};

// 模型数据导出
export const exportModelData = params => {
  return service({
    url: `${USER_API}/statistic/model/export`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

/**
 * 应用统计接口
 */

// 获取应用下拉列表
export const getAppSelect = params => {
  return service({
    url: `${USER_API}/statistic/app/select`,
    method: 'get',
    params,
  });
};

// 获取应用统计数据
export const getAppData = params => {
  return service({
    url: `${USER_API}/statistic/app`,
    method: 'get',
    params,
  });
};

// 获取应用统计列表
export const fetchAppList = params => {
  return service({
    url: `${USER_API}/statistic/app/list`,
    method: 'get',
    params,
  });
};

// 应用数据导出
export const exportAppData = params => {
  return service({
    url: `${USER_API}/statistic/app/export`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

/**
 * API统计接口
 */

// 获取API下拉列表
export const getApiSelect = params => {
  return service({
    url: `${USER_API}/statistic/api/select`,
    method: 'get',
    params,
  });
};

// 获取API路径列表
export const getApiRoutes = params => {
  return service({
    url: `${USER_API}/statistic/api/routes`,
    method: 'get',
    params,
  });
};

// 获取API统计数据
export const getApiData = data => {
  return service({
    url: `${USER_API}/statistic/api`,
    method: 'post',
    data,
  });
};

// 获取模型列表
export const fetchApiList = data => {
  const type = data.type;
  delete data.type;
  return service({
    url: `${USER_API}/statistic/api/${type || 'list'}`,
    method: 'post',
    data,
  });
};

// 模型数据导出
export const exportApiData = (data, type) => {
  return service({
    url: `${USER_API}/statistic/api/export/${type}`,
    method: 'post',
    data,
    responseType: 'blob',
  });
};
