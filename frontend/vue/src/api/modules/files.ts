/** 文件管理 API */
import api from '../client'
import type {
  FileListResponse,
  QuickDirsResponse,
  FileReadResponse,
  TrashListResponse,
  ShareLink,
} from '@/types/file'

export function listFiles(path: string, dirsOnly = false): Promise<FileListResponse> {
  return api.get<FileListResponse>('/api/files', { params: { path, dirsOnly: dirsOnly ? 'true' : undefined } })
}

export function getQuickDirs(): Promise<QuickDirsResponse> {
  return api.get<QuickDirsResponse>('/api/files/quick-dirs')
}

export function readFileInfo(path: string, name: string): Promise<FileReadResponse> {
  return api.get<FileReadResponse>('/api/files/read', { params: { path, name } })
}

export function saveFileContent(path: string, name: string, content: string): Promise<void> {
  return api.post('/api/files/save', { path, name, content })
}

export function mkdir(fullPath: string): Promise<void> {
  return api.post('/api/files/mkdir', { path: fullPath })
}

export function renameFile(path: string, oldName: string, newName: string): Promise<void> {
  return api.post('/api/files/rename', { path, oldName, newName })
}

export function deleteFile(path: string, name: string): Promise<{ trash_id: string }> {
  return api.post('/api/files/delete', { path, name })
}

export function moveFile(srcPath: string, srcName: string, dstPath: string): Promise<void> {
  return api.post('/api/files/move', { srcPath, srcName, dstPath })
}

export function copyFile(srcPath: string, srcName: string, dstPath?: string, dstName?: string): Promise<void> {
  return api.post('/api/files/copy', { srcPath, srcName, dstPath, dstName })
}

export function touchFile(path: string, name: string): Promise<void> {
  return api.post('/api/files/touch', { path, name })
}

export function compressFiles(path: string, files: string[], format = 'zip', target = 'archive'): Promise<{ file: string; path: string }> {
  return api.post('/api/files/compress', { path, files, format, target })
}

export function extractFile(path: string, name: string): Promise<void> {
  return api.post('/api/files/extract', { path, name })
}

export function shareFile(path: string, name: string, durationHours: number, password?: string): Promise<{ token: string; url: string; expiresAt: string; fileName: string; fileSize: number }> {
  return api.post('/api/files/share', { path, name, duration: durationHours, password })
}

export function listShares(): Promise<{ links: ShareLink[] }> {
  return api.get('/api/files/share/list')
}

export function deleteShare(token: string): Promise<void> {
  return api.delete('/api/files/share/delete', { data: { token } })
}

// 直接上传（multipart）
export function uploadToPath(file: File, path: string, relativePath?: string): Promise<{ filename: string }> {
  const form = new FormData()
  form.append('file', file)
  form.append('path', path)
  if (relativePath) form.append('relativePath', relativePath)
  return api.raw.post('/api/files/upload/path', form).then((res: { data: { code: number; message: string; data?: { filename: string } } }) => {
    if (res.data.code !== 200) throw new Error(res.data.message)
    return res.data.data ?? { filename: file.name }
  })
}

// 回收站
export function listTrash(): Promise<TrashListResponse> {
  return api.get<TrashListResponse>('/api/files/trash/list')
}

export function restoreTrash(id: string): Promise<void> {
  return api.post('/api/files/trash/restore', { id })
}

export function deleteTrash(id: string): Promise<void> {
  return api.post('/api/files/trash/delete', { id })
}

export function clearTrash(): Promise<void> {
  return api.post('/api/files/trash/clear')
}

// 脚本执行
export function runScript(path: string, background = false): Promise<unknown> {
  return api.post('/api/files/run', { path, background })
}

export function listProcesses(): Promise<{ processes: unknown[] }> {
  return api.get('/api/files/processes')
}

export function stopProcess(id: string): Promise<void> {
  return api.post('/api/files/processes/stop', { id })
}
