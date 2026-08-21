package jsonrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

const librarySchemaVersion = 2

type libraryState struct {
	SchemaVersion int                     `json:"schema_version"`
	NextID        int64                   `json:"next_id"`
	NextVersionID int64                   `json:"next_version_id"`
	Items         []models.LibraryItem    `json:"items"`
	Versions      []models.LibraryVersion `json:"versions"`
}

type LibraryRepository struct{ store *base.JSONStore }

// loadLibrary 在存储层中读取并整理所需数据。
func loadLibrary(tx *base.JSONTx) (libraryState, error) {
	state := libraryState{NextID: 1, NextVersionID: 1, Items: []models.LibraryItem{}, Versions: []models.LibraryVersion{}}
	if err := tx.Load("library.json", &state); err != nil {
		return state, err
	}
	if state.NextID < 1 {
		state.NextID = 1
	}
	if state.NextVersionID < 1 {
		state.NextVersionID = 1
	}
	return state, nil
}

// saveLibrary 在存储层中创建或更新相应状态。
func saveLibrary(tx *base.JSONTx, state libraryState) error { return tx.Save("library.json", state) }

// List 在存储层中读取并整理所需数据。
func (r *LibraryRepository) List(ctx context.Context, filter base.LibraryFilter) ([]models.LibraryItem, error) {
	result := []models.LibraryItem{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		itemsByID := make(map[int64]models.LibraryItem, len(state.Items))
		for _, item := range state.Items {
			itemsByID[item.ID] = item
		}
		q := strings.ToLower(strings.TrimSpace(filter.Query))
		for _, item := range state.Items {
			if filter.Trashed != (item.DeletedAt != nil) {
				continue
			}
			if filter.ParentID != nil && (item.ParentID == nil || *item.ParentID != *filter.ParentID) {
				continue
			}
			if filter.Trashed && filter.ParentID == nil && hasTrashedLibraryAncestor(item, itemsByID) {
				continue
			}
			if filter.ParentID == nil && !filter.All && filter.Query == "" && !filter.Trashed && !filter.ReviewOnly && item.ParentID != nil {
				continue
			}
			if filter.Kind != "" && filter.Kind != "all" && item.Kind != filter.Kind {
				continue
			}
			if filter.Tag != "" && !containsTag(item.Tags, filter.Tag) {
				continue
			}
			if filter.ReviewOnly && !item.ReviewEnabled {
				continue
			}
			if filter.DueOnly && (!item.ReviewEnabled || item.NextReview == "" || item.NextReview > time.Now().Format("2006-01-02")) {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(item.Name+" "+strings.Join(item.Tags, " ")), q) {
				if item.Kind != "note" || item.BlobHash == "" {
					continue
				}
				body, _ := base.ReadBlob(item.BlobHash)
				if !strings.Contains(strings.ToLower(string(body)), q) {
					continue
				}
			}
			result = append(result, item)
		}
		return nil
	})
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Pinned != result[j].Pinned {
			return result[i].Pinned
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, err
}

// hasTrashedLibraryAncestor 在回收站视图中保持已删除文件夹树的完整性：只列出被删除树的根节点；父节点仍有效而单独删除的笔记仍会显示并可恢复。
// hasTrashedLibraryAncestor 判断条目是否位于已删除父目录的子树中。
func hasTrashedLibraryAncestor(item models.LibraryItem, itemsByID map[int64]models.LibraryItem) bool {
	parentID := item.ParentID
	for parentID != nil {
		parent, exists := itemsByID[*parentID]
		if !exists {
			return false
		}
		if parent.DeletedAt != nil {
			return true
		}
		parentID = parent.ParentID
	}
	return false
}

// Get 在存储层中读取并整理所需数据。
func (r *LibraryRepository) Get(ctx context.Context, id int64) (models.LibraryItem, error) {
	var result models.LibraryItem
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		idx := itemIndex(state.Items, id)
		if idx < 0 {
			return fmt.Errorf("资料不存在")
		}
		result = state.Items[idx]
		return nil
	})
	return result, err
}

// Create 在存储层中创建或更新相应状态。
func (r *LibraryRepository) Create(ctx context.Context, req models.CreateLibraryItemRequest, content []byte) (models.LibraryItem, error) {
	var result models.LibraryItem
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return result, fmt.Errorf("名称不能为空")
	}
	if !validKind(req.Kind) {
		return result, fmt.Errorf("无效资料类型")
	}
	var hash string
	var size int64
	var err error
	if req.Kind != "folder" {
		hash, size, err = base.StoreBlob(bytes.NewReader(content))
		if err != nil {
			return result, err
		}
	}
	err = r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		if e = validateParent(state.Items, req.ParentID, 0); e != nil {
			return e
		}
		name = uniqueName(state.Items, req.ParentID, name, 0)
		now := time.Now().UTC()
		result = models.LibraryItem{ID: state.NextID, ParentID: req.ParentID, Kind: req.Kind, Name: name, MimeType: req.MimeType, Size: size, Tags: normalizeTags(req.Tags), ErrorProblemID: req.ErrorProblemID, BlobHash: hash, ReviewEnabled: req.ReviewEnabled, CreatedAt: now, UpdatedAt: now}
		if req.ReviewEnabled {
			result.NextReview = now.Format("2006-01-02")
		}
		state.NextID++
		if req.Kind != "folder" {
			result.CurrentVersion = 1
			state.Versions = append(state.Versions, models.LibraryVersion{ID: state.NextVersionID, ItemID: result.ID, Version: 1, BlobHash: hash, Size: size, CreatedAt: now})
			state.NextVersionID++
		}
		state.Items = append(state.Items, result)
		return saveLibrary(tx, state)
	})
	return result, err
}

// Update 在存储层中创建或更新相应状态。
func (r *LibraryRepository) Update(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error) {
	var result models.LibraryItem
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		idx := itemIndex(state.Items, id)
		if idx < 0 {
			return fmt.Errorf("资料不存在")
		}
		item := &state.Items[idx]
		if req.Name != nil {
			n := strings.TrimSpace(*req.Name)
			if n == "" {
				return fmt.Errorf("名称不能为空")
			}
			item.Name = uniqueName(state.Items, item.ParentID, n, id)
		}
		if req.Tags != nil {
			item.Tags = normalizeTags(*req.Tags)
		}
		if req.Pinned != nil {
			item.Pinned = *req.Pinned
		}
		if req.ParentSet || req.ParentID != nil {
			if e = validateParent(state.Items, req.ParentID, id); e != nil {
				return e
			}
			item.ParentID = req.ParentID
		}
		if req.ReviewEnabled != nil {
			item.ReviewEnabled = *req.ReviewEnabled
			if item.ReviewEnabled && item.NextReview == "" {
				item.NextReview = time.Now().Format("2006-01-02")
			}
		}
		item.UpdatedAt = time.Now().UTC()
		result = *item
		return saveLibrary(tx, state)
	})
	return result, err
}

// SaveContent 在存储层中创建或更新相应状态。
func (r *LibraryRepository) SaveContent(ctx context.Context, id int64, content []byte, baseVersion int, checkpoint, force bool) (models.LibraryItem, error) {
	var result models.LibraryItem
	hash, size, err := base.StoreBlob(bytes.NewReader(content))
	if err != nil {
		return result, err
	}
	err = r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		idx := itemIndex(state.Items, id)
		if idx < 0 {
			return fmt.Errorf("资料不存在")
		}
		item := &state.Items[idx]
		if item.Kind == "folder" {
			return fmt.Errorf("该资料不能保存正文")
		}
		if !force && baseVersion != item.CurrentVersion {
			return fmt.Errorf("版本冲突")
		}
		if hash == item.BlobHash {
			if checkpoint && !hasVersion(state.Versions, id, item.CurrentVersion) {
				state.Versions = append(state.Versions, models.LibraryVersion{ID: state.NextVersionID, ItemID: id, Version: item.CurrentVersion, BlobHash: hash, Size: size, CreatedAt: time.Now().UTC()})
				state.NextVersionID++
				trimVersions(&state, id, 50)
				result = *item
				return saveLibrary(tx, state)
			}
			result = *item
			return nil
		}
		item.CurrentVersion++
		item.BlobHash = hash
		item.Size = size
		item.UpdatedAt = time.Now().UTC()
		if checkpoint {
			state.Versions = append(state.Versions, models.LibraryVersion{ID: state.NextVersionID, ItemID: id, Version: item.CurrentVersion, BlobHash: hash, Size: size, CreatedAt: item.UpdatedAt})
			state.NextVersionID++
			trimVersions(&state, id, 50)
		}
		result = *item
		return saveLibrary(tx, state)
	})
	return result, err
}

// ReadContent 在存储层中读取并整理所需数据。
func (r *LibraryRepository) ReadContent(ctx context.Context, id int64) ([]byte, models.LibraryItem, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, item, err
	}
	if item.BlobHash == "" {
		return []byte{}, item, nil
	}
	body, err := base.ReadBlob(item.BlobHash)
	return body, item, err
}

// Trash 在存储层中删除、清理或撤销相应状态。
func (r *LibraryRepository) Trash(ctx context.Context, id int64) error {
	return r.setTrash(ctx, id, true)
}

// Restore 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Restore(ctx context.Context, id int64) (models.LibraryItem, error) {
	var out models.LibraryItem
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		i := itemIndex(s.Items, id)
		if i < 0 {
			return fmt.Errorf("资料不存在")
		}
		s.Items[i].DeletedAt = nil
		s.Items[i].ParentID = s.Items[i].OriginalParent
		s.Items[i].OriginalParent = nil
		s.Items[i].Name = uniqueName(s.Items, s.Items[i].ParentID, s.Items[i].Name, id)
		s.Items[i].UpdatedAt = time.Now().UTC()
		for n := range s.Items {
			if isDescendant(s.Items, s.Items[n].ID, id) {
				s.Items[n].DeletedAt = nil
				s.Items[n].OriginalParent = nil
				s.Items[n].UpdatedAt = time.Now().UTC()
			}
		}
		out = s.Items[i]
		return saveLibrary(tx, s)
	})
	return out, err
}

// setTrash 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) setTrash(ctx context.Context, id int64, trash bool) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		i := itemIndex(s.Items, id)
		if i < 0 {
			return fmt.Errorf("资料不存在")
		}
		now := time.Now().UTC()
		s.Items[i].OriginalParent = s.Items[i].ParentID
		s.Items[i].DeletedAt = &now
		for n := range s.Items {
			if isDescendant(s.Items, s.Items[n].ID, id) {
				s.Items[n].DeletedAt = &now
			}
		}
		return saveLibrary(tx, s)
	})
}

// Purge 在存储层中删除、清理或撤销相应状态。
func (r *LibraryRepository) Purge(ctx context.Context, id int64) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		keep := s.Items[:0]
		ids := map[int64]bool{id: true}
		for _, x := range s.Items {
			if isDescendant(s.Items, x.ID, id) {
				ids[x.ID] = true
			}
		}
		for _, x := range s.Items {
			if !ids[x.ID] {
				keep = append(keep, x)
			}
		}
		s.Items = keep
		vk := s.Versions[:0]
		for _, v := range s.Versions {
			if !ids[v.ItemID] {
				vk = append(vk, v)
			}
		}
		s.Versions = vk
		return saveLibrary(tx, s)
	})
}

// Batch 使 JSON 实现与 PostgreSQL 语义一致：保存状态前先校验全部输入，因此失败的选中项不会被部分应用。
// Batch 原子执行资料库条目的批量移动、恢复或删除操作。
func (r *LibraryRepository) Batch(ctx context.Context, action string, ids []int64, parentID *int64) error {
	ids = uniqueLibraryIDs(ids)
	if len(ids) == 0 {
		return fmt.Errorf("至少选择一项资料")
	}
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		selected := make(map[int64]models.LibraryItem, len(ids))
		for _, id := range ids {
			index := itemIndex(state.Items, id)
			if index < 0 {
				return fmt.Errorf("所选资料不存在或无权操作")
			}
			selected[id] = state.Items[index]
		}
		roots := batchLibraryRoots(ids, state.Items)
		switch action {
		case "trash":
			for _, id := range roots {
				if selected[id].DeletedAt != nil {
					return fmt.Errorf("所选资料已在回收站中")
				}
			}
			now := time.Now().UTC()
			for index := range state.Items {
				for _, rootID := range roots {
					if state.Items[index].ID == rootID || isDescendant(state.Items, state.Items[index].ID, rootID) {
						if state.Items[index].OriginalParent == nil {
							state.Items[index].OriginalParent = state.Items[index].ParentID
						}
						state.Items[index].DeletedAt = &now
						state.Items[index].UpdatedAt = now
						break
					}
				}
			}
		case "restore":
			for _, id := range roots {
				if selected[id].DeletedAt == nil {
					return fmt.Errorf("所选资料不在回收站中")
				}
			}
			now := time.Now().UTC()
			for index := range state.Items {
				for _, rootID := range roots {
					if state.Items[index].ID == rootID || isDescendant(state.Items, state.Items[index].ID, rootID) {
						if state.Items[index].OriginalParent != nil {
							state.Items[index].ParentID = state.Items[index].OriginalParent
						}
						state.Items[index].OriginalParent = nil
						state.Items[index].DeletedAt = nil
						state.Items[index].UpdatedAt = now
						break
					}
				}
			}
		case "purge":
			for _, id := range roots {
				if selected[id].DeletedAt == nil {
					return fmt.Errorf("只能永久删除回收站中的资料")
				}
			}
			removed := make(map[int64]bool)
			for _, item := range state.Items {
				for _, rootID := range roots {
					if item.ID == rootID || isDescendant(state.Items, item.ID, rootID) {
						removed[item.ID] = true
						break
					}
				}
			}
			items := state.Items[:0]
			for _, item := range state.Items {
				if !removed[item.ID] {
					items = append(items, item)
				}
			}
			state.Items = items
			versions := state.Versions[:0]
			for _, version := range state.Versions {
				if !removed[version.ItemID] {
					versions = append(versions, version)
				}
			}
			state.Versions = versions
		case "move":
			for _, id := range roots {
				if selected[id].DeletedAt != nil {
					return fmt.Errorf("不能移动回收站中的资料")
				}
				if err = validateParent(state.Items, parentID, id); err != nil {
					return err
				}
			}
			for _, id := range roots {
				index := itemIndex(state.Items, id)
				state.Items[index].ParentID = parentID
				state.Items[index].Name = uniqueName(state.Items, parentID, state.Items[index].Name, id)
				state.Items[index].UpdatedAt = time.Now().UTC()
			}
		default:
			return fmt.Errorf("不支持的批量操作")
		}
		return saveLibrary(tx, state)
	})
}

// uniqueLibraryIDs 在存储层中完成本文件定义的局部处理。
func uniqueLibraryIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique
}

// batchLibraryRoots 在存储层中完成本文件定义的局部处理。
func batchLibraryRoots(ids []int64, items []models.LibraryItem) []int64 {
	roots := make([]int64, 0, len(ids))
	for _, id := range ids {
		child := false
		for _, candidate := range ids {
			if id != candidate && isDescendant(items, id, candidate) {
				child = true
				break
			}
		}
		if !child {
			roots = append(roots, id)
		}
	}
	return roots
}

// Duplicate 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Duplicate(ctx context.Context, id int64, parentID *int64) (models.LibraryItem, error) {
	body, item, err := r.ReadContent(ctx, id)
	if err != nil {
		return models.LibraryItem{}, err
	}
	if item.Kind == "folder" {
		return r.Create(ctx, models.CreateLibraryItemRequest{ParentID: parentID, Kind: "folder", Name: item.Name, Tags: item.Tags}, nil)
	}
	return r.Create(ctx, models.CreateLibraryItemRequest{ParentID: parentID, Kind: item.Kind, Name: item.Name, MimeType: item.MimeType, Tags: item.Tags, ReviewEnabled: item.ReviewEnabled}, body)
}

// Versions 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Versions(ctx context.Context, id int64) ([]models.LibraryVersion, error) {
	out := []models.LibraryVersion{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		for _, v := range s.Versions {
			if v.ItemID == id {
				out = append(out, v)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Version > out[j].Version })
		return nil
	})
	return out, err
}

// RestoreVersion 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) RestoreVersion(ctx context.Context, id, versionID int64) (models.LibraryItem, error) {
	var content []byte
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		for _, v := range s.Versions {
			if v.ID == versionID && v.ItemID == id {
				content, e = base.ReadBlob(v.BlobHash)
				return e
			}
		}
		return fmt.Errorf("版本不存在")
	})
	if err != nil {
		return models.LibraryItem{}, err
	}
	item, err := r.Get(ctx, id)
	if err != nil {
		return item, err
	}
	// 恢复会改变当前内容，但不是用户新建的检查点；为冲突检测保持修订令牌单调递增，同时不改变可见版本历史。
	return r.SaveContent(ctx, id, content, item.CurrentVersion, false, true)
}

// EnsureLegacy 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) EnsureLegacy(ctx context.Context, errs []models.ErrorProblem, subjects []string) error {
	legacy := make(map[int]struct {
		body []byte
		hash string
		size int64
	})
	for _, problem := range errs {
		body := []byte(legacyErrorMarkdown(problem))
		hash, size, err := base.StoreBlob(bytes.NewReader(body))
		if err != nil {
			return err
		}
		legacy[problem.ID] = struct {
			body []byte
			hash string
			size int64
		}{body: body, hash: hash, size: size}
	}
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, e := loadLibrary(tx)
		if e != nil {
			return e
		}
		now := time.Now().UTC()
		for _, problem := range errs {
			data := legacy[problem.ID]
			eid := problem.ID
			name := strings.TrimSpace(problem.Title)
			if name == "" {
				name = fmt.Sprintf("复习笔记 #%d", problem.ID)
			}
			linked := false
			for i := range s.Items {
				if s.Items[i].ErrorProblemID != nil && *s.Items[i].ErrorProblemID == problem.ID {
					item := &s.Items[i]
					if item.Kind == "note" {
						linked = true
						break
					}
					item.Kind = "note"
					item.ParentID = nil
					item.Name = uniqueName(s.Items, nil, name, item.ID)
					item.Tags = mergeTags(problem.Tags, problem.ReasonTags, []string{problem.Subject})
					item.BlobHash, item.Size = data.hash, data.size
					item.ReviewEnabled = true
					item.ReviewCount, item.ReviewStage, item.NextReview = problem.ReviewCount, problem.ReviewStage, problem.NextReview
					item.LastReview = parseLegacyReview(problem.LastReview)
					if item.NextReview == "" {
						item.NextReview = now.Format("2006-01-02")
					}
					if item.CurrentVersion < 1 {
						item.CurrentVersion = 1
					}
					item.UpdatedAt = now
					if !hasVersion(s.Versions, item.ID, item.CurrentVersion) {
						s.Versions = append(s.Versions, models.LibraryVersion{ID: s.NextVersionID, ItemID: item.ID, Version: item.CurrentVersion, BlobHash: data.hash, Size: data.size, CreatedAt: now})
						s.NextVersionID++
					}
					linked = true
					break
				}
			}
			if linked {
				continue
			}
			item := models.LibraryItem{ID: s.NextID, Kind: "note", Name: uniqueName(s.Items, nil, name, 0), MimeType: "text/markdown; charset=utf-8", Size: data.size, Tags: mergeTags(problem.Tags, problem.ReasonTags, []string{problem.Subject}), ErrorProblemID: &eid, BlobHash: data.hash, CurrentVersion: 1, ReviewEnabled: true, ReviewCount: problem.ReviewCount, ReviewStage: problem.ReviewStage, LastReview: parseLegacyReview(problem.LastReview), NextReview: problem.NextReview, CreatedAt: now, UpdatedAt: now}
			if item.NextReview == "" {
				item.NextReview = now.Format("2006-01-02")
			}
			s.Items = append(s.Items, item)
			s.Versions = append(s.Versions, models.LibraryVersion{ID: s.NextVersionID, ItemID: item.ID, Version: 1, BlobHash: data.hash, Size: data.size, CreatedAt: now})
			s.NextVersionID++
			s.NextID++
		}
		removeEmptySystemFolders(&s, subjects)
		s.SchemaVersion = librarySchemaVersion
		return saveLibrary(tx, s)
	})
}

// ListTags 在存储层中读取并整理所需数据。
func (r *LibraryRepository) ListTags(ctx context.Context) ([]string, error) {
	set := map[string]string{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		s, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		for _, item := range s.Items {
			if item.DeletedAt != nil {
				continue
			}
			for _, tag := range item.Tags {
				set[strings.ToLower(tag)] = tag
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for _, tag := range set {
		out = append(out, tag)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out, nil
}

// DueReviews 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) DueReviews(ctx context.Context, day time.Time) ([]models.LibraryItem, error) {
	items, err := r.List(ctx, base.LibraryFilter{ReviewOnly: true})
	if err != nil {
		return nil, err
	}
	today := day.Format("2006-01-02")
	out := make([]models.LibraryItem, 0, len(items))
	for _, item := range items {
		if item.NextReview == "" || item.NextReview <= today {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NextReview == out[j].NextReview {
			return out[i].ID < out[j].ID
		}
		return out[i].NextReview < out[j].NextReview
	})
	return out, nil
}

// Review 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Review(ctx context.Context, id int64, reviewedAt time.Time, intervals []int) (models.LibraryItem, error) {
	return r.ReviewWithRating(ctx, id, reviewedAt, "good")
}

// ReviewWithRating 在存储层中执行当前数据访问或局部处理。
func (r *LibraryRepository) ReviewWithRating(ctx context.Context, id int64, reviewedAt time.Time, rating string) (models.LibraryItem, error) {
	var out models.LibraryItem
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		i := itemIndex(s.Items, id)
		if i < 0 || s.Items[i].DeletedAt != nil || !s.Items[i].ReviewEnabled {
			return fmt.Errorf("复习笔记不存在")
		}
		item := &s.Items[i]
		item.ReviewStage, item.ReviewCount, item.NextReview = models.NextReview(item.ReviewStage, item.ReviewCount, rating, reviewedAt)
		item.LastReview = &reviewedAt
		item.UpdatedAt = reviewedAt.UTC()
		out = *item
		return saveLibrary(tx, s)
	})
	return out, err
}

// legacyErrorMarkdown 在存储层中完成本文件定义的局部处理。
func legacyErrorMarkdown(problem models.ErrorProblem) string {
	return "## 题目\n\n" + strings.TrimSpace(problem.Question) + "\n\n## 错解\n\n" + strings.TrimSpace(problem.Wrong) + "\n\n## 正解\n\n" + strings.TrimSpace(problem.Correct) + "\n\n## 错因\n\n" + strings.TrimSpace(problem.Reason) + "\n"
}

// parseLegacyReview 在存储层中解析外部输入为内部数据。
func parseLegacyReview(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, *value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

// mergeTags 在存储层中完成本文件定义的局部处理。
func mergeTags(groups ...[]string) []string {
	all := []string{}
	for _, group := range groups {
		all = append(all, group...)
	}
	return normalizeTags(all)
}

// containsTag 在存储层中完成本文件定义的局部处理。
func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

// removeEmptySystemFolders 在存储层中删除、清理或撤销相应状态。
func removeEmptySystemFolders(s *libraryState, subjects []string) {
	subjectNames := map[string]bool{}
	for _, subject := range subjects {
		subjectNames[strings.TrimSpace(subject)] = true
	}
	changed := true
	for changed {
		changed = false
		legacyRoots := map[int64]bool{}
		for _, item := range s.Items {
			if item.Kind == "folder" && item.ParentID == nil && item.Name == "错题库" {
				legacyRoots[item.ID] = true
			}
		}
		for i := len(s.Items) - 1; i >= 0; i-- {
			item := s.Items[i]
			isRoot := legacyRoots[item.ID]
			isLegacySubject := item.ParentID != nil && legacyRoots[*item.ParentID] && subjectNames[item.Name]
			if item.Kind != "folder" || (!isRoot && !isLegacySubject) {
				continue
			}
			hasChild := false
			for _, child := range s.Items {
				if child.ParentID != nil && *child.ParentID == item.ID {
					hasChild = true
					break
				}
			}
			if hasChild {
				continue
			}
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			changed = true
		}
	}
}

// Cleanup 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Cleanup(ctx context.Context, before time.Time) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		s, err := loadLibrary(tx)
		if err != nil {
			return err
		}
		ids := map[int64]bool{}
		for _, item := range s.Items {
			if item.DeletedAt != nil && item.DeletedAt.Before(before) {
				ids[item.ID] = true
			}
		}
		if len(ids) == 0 {
			return nil
		}
		items := s.Items[:0]
		for _, item := range s.Items {
			if !ids[item.ID] {
				items = append(items, item)
			}
		}
		s.Items = items
		versions := s.Versions[:0]
		for _, version := range s.Versions {
			if !ids[version.ItemID] {
				versions = append(versions, version)
			}
		}
		s.Versions = versions
		return saveLibrary(tx, s)
	})
}

// itemIndex 在存储层中完成本文件定义的局部处理。
func itemIndex(items []models.LibraryItem, id int64) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

// validKind 在存储层中完成本文件定义的局部处理。
func validKind(v string) bool { return v == "folder" || v == "note" || v == "file" }

// normalizeTags 在存储层中构造、编码或标准化数据。
func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" && !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			out = append(out, t)
		}
	}
	return out
}

// sameParent 在存储层中完成本文件定义的局部处理。
func sameParent(a, b *int64) bool { return a == nil && b == nil || (a != nil && b != nil && *a == *b) }

// uniqueName 在存储层中完成本文件定义的局部处理。
func uniqueName(items []models.LibraryItem, parent *int64, name string, except int64) string {
	baseName := name
	candidate := name
	for n := 2; ; n++ {
		used := false
		for _, x := range items {
			if x.ID != except && x.DeletedAt == nil && sameParent(x.ParentID, parent) && strings.EqualFold(x.Name, candidate) {
				used = true
				break
			}
		}
		if !used {
			return candidate
		}
		candidate = fmt.Sprintf("%s (%d)", baseName, n)
	}
}

// validateParent 在存储层中校验输入或判断当前条件。
func validateParent(items []models.LibraryItem, parent *int64, self int64) error {
	if parent == nil {
		return nil
	}
	i := itemIndex(items, *parent)
	if i < 0 || items[i].Kind != "folder" || items[i].DeletedAt != nil {
		return fmt.Errorf("目标文件夹不存在")
	}
	if *parent == self || isDescendant(items, *parent, self) {
		return fmt.Errorf("不能移动到自身或子文件夹")
	}
	return nil
}

// isDescendant 在存储层中校验输入或判断当前条件。
func isDescendant(items []models.LibraryItem, id, ancestor int64) bool {
	if ancestor == 0 {
		return false
	}
	seen := map[int64]bool{}
	for id != 0 && !seen[id] {
		seen[id] = true
		i := itemIndex(items, id)
		if i < 0 || items[i].ParentID == nil {
			return false
		}
		if *items[i].ParentID == ancestor {
			return true
		}
		id = *items[i].ParentID
	}
	return false
}

// trimVersions 在存储层中完成本文件定义的局部处理。
func trimVersions(s *libraryState, id int64, max int) {
	indices := []int{}
	for i, v := range s.Versions {
		if v.ItemID == id {
			indices = append(indices, i)
		}
	}
	if len(indices) <= max {
		return
	}
	drop := len(indices) - max
	keep := s.Versions[:0]
	for i, v := range s.Versions {
		remove := false
		for _, idx := range indices[:drop] {
			if i == idx {
				remove = true
				break
			}
		}
		if !remove {
			keep = append(keep, v)
		}
	}
	s.Versions = keep
}

// hasVersion 在存储层中校验输入或判断当前条件。
func hasVersion(items []models.LibraryVersion, itemID int64, version int) bool {
	for _, v := range items {
		if v.ItemID == itemID && v.Version == version {
			return true
		}
	}
	return false
}

var _ base.LibraryRepository = (*LibraryRepository)(nil)
var _ = errors.Is
