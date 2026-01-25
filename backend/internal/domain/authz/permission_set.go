package authz

// PermissionSet は権限のセットを表す型
type PermissionSet struct {
	permissions map[Permission]bool
}

// NewPermissionSet は新しいPermissionSetを生成します
func NewPermissionSet(perms ...Permission) *PermissionSet {
	ps := &PermissionSet{
		permissions: make(map[Permission]bool),
	}
	for _, p := range perms {
		ps.permissions[p] = true
	}
	return ps
}

// EmptyPermissionSet は空のPermissionSetを生成します
func EmptyPermissionSet() *PermissionSet {
	return &PermissionSet{
		permissions: make(map[Permission]bool),
	}
}

// FromRole はロールからPermissionSetを生成します
func FromRole(role Role) *PermissionSet {
	return NewPermissionSet(role.Permissions()...)
}

// Add は権限を追加します
func (ps *PermissionSet) Add(perm Permission) {
	ps.permissions[perm] = true
}

// AddAll は複数の権限を追加します
func (ps *PermissionSet) AddAll(perms ...Permission) {
	for _, p := range perms {
		ps.permissions[p] = true
	}
}

// AddFromRole はロールの権限を追加します
func (ps *PermissionSet) AddFromRole(role Role) {
	ps.AddAll(role.Permissions()...)
}

// Remove は権限を削除します
func (ps *PermissionSet) Remove(perm Permission) {
	delete(ps.permissions, perm)
}

// Has は権限を持っているかを判定します
func (ps *PermissionSet) Has(perm Permission) bool {
	return ps.permissions[perm]
}

// HasAny はいずれかの権限を持っているかを判定します
func (ps *PermissionSet) HasAny(perms ...Permission) bool {
	for _, p := range perms {
		if ps.permissions[p] {
			return true
		}
	}
	return false
}

// HasAll は全ての権限を持っているかを判定します
func (ps *PermissionSet) HasAll(perms ...Permission) bool {
	for _, p := range perms {
		if !ps.permissions[p] {
			return false
		}
	}
	return true
}

// List は権限の一覧を返します
func (ps *PermissionSet) List() []Permission {
	perms := make([]Permission, 0, len(ps.permissions))
	for p := range ps.permissions {
		perms = append(perms, p)
	}
	return perms
}

// IsEmpty は空かどうかを判定します
func (ps *PermissionSet) IsEmpty() bool {
	return len(ps.permissions) == 0
}

// Size は権限の数を返します
func (ps *PermissionSet) Size() int {
	return len(ps.permissions)
}

// Union は2つのPermissionSetの和集合を返します
func (ps *PermissionSet) Union(other *PermissionSet) *PermissionSet {
	result := NewPermissionSet(ps.List()...)
	result.AddAll(other.List()...)
	return result
}

// Intersection は2つのPermissionSetの積集合を返します
func (ps *PermissionSet) Intersection(other *PermissionSet) *PermissionSet {
	result := EmptyPermissionSet()
	for p := range ps.permissions {
		if other.Has(p) {
			result.Add(p)
		}
	}
	return result
}

// Clone は複製を返します
func (ps *PermissionSet) Clone() *PermissionSet {
	return NewPermissionSet(ps.List()...)
}
