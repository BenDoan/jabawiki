var app = angular.module('wiki', ['ngRoute', 'angular-loading-bar']).
    config(['$routeProvider', '$locationProvider',
    function($routeProvider, $locationProvider){
        $routeProvider.
            when('/', {
                redirectTo: '/w/Home'
            }).
            when('/login', {
                templateUrl: 'partials/login.html',
                controller: 'LoginCtrl'
            }).
            when('/w/:title/edit', {
                templateUrl: 'partials/edit.html',
                controller: 'ArticleEditCtrl'
            }).
            when('/w/:title/history', {
                templateUrl: 'partials/history.html',
                controller: 'HistoryCtrl'
            }).
            when('/w/:title', {
                templateUrl: 'partials/view.html',
                controller: 'ArticleViewCtrl'
            }).
            when('/index', {
                templateUrl: 'partials/index.html',
                controller: 'IndexCtrl'
            }).
            when('/profile', {
                templateUrl: 'partials/profile.html',
                controller: 'ProfileCtrl'
            }).
            when('/uploadimage', {
                templateUrl: 'partials/uploadimage.html',
                controller: 'UploadImageCtrl'
            }).
            otherwise({
                redirectTo: '/'
            });

        $locationProvider.html5Mode(true);

}]);

